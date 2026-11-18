"""Converts the Donut checkpoints used by the built-in "donut" recognition provider to the ONNX files downloaded by ezBookkeeping.

Requirements (CPU only is fine):

    pip install --extra-index-url https://download.pytorch.org/whl/cpu torch==2.15.0+cpu transformers==4.57.6 \
        optimum==2.1.0 optimum-onnx==0.1.0 onnx==1.23.2 onnx_ir onnxruntime==1.32.0 safetensors huggingface-hub

Usage:

    python convert_models.py <output dir>

The output directory contains one folder per model (receipts, sroie) with donut.json, tokenizer.json, encoder.onnx and
decoder.onnx. Update the sizes and SHA-256 checksums in pkg/donut/model_store.go after converting.

Quantization (chosen by comparing the output with the full precision models on test receipts):
* encoder: dynamic 8-bit (per tensor), the result is identical to full precision
* decoder: 8-bit weights for MatMul with block-wise quantized activations (MatMulNBits, accuracy level 4) and 8-bit embeddings.
  Plain dynamic 8-bit quantization of the decoder changes the read text, and per-channel quantization breaks it entirely.
"""

import glob
import json
import os
import shutil
import subprocess
import sys
import tempfile

import torch
from huggingface_hub import snapshot_download
from onnxruntime.quantization import QuantType, quantize_dynamic
from onnxruntime.quantization.matmul_nbits_quantizer import DefaultWeightOnlyQuantConfig, MatMulNBitsQuantizer
from safetensors.torch import load_file
from transformers import DonutProcessor, VisionEncoderDecoderModel

MODELS = [
    # name, Hugging Face checkpoint, revision, task prompt
    ("receipts", "mychen76/invoice-and-receipts_donut_v1", "main", "<s_receipt>"),
    ("sroie", "unstructuredio/donut-base-sroie", "main", "<s>"),
]
MAX_LENGTH = 768


def convert(name, repo, revision, task_prompt, output_dir, work_dir):
    source = snapshot_download(repo, revision=revision)
    model = VisionEncoderDecoderModel.from_pretrained(source, low_cpu_mem_usage=False)
    embeddings = model.decoder.get_input_embeddings()

    # some checkpoints only store decoder.lm_head.weight and rely on weight tying, which recent transformers versions skip
    if embeddings.weight.device.type == "meta":
        weights = load_file(glob.glob(os.path.join(source, "*.safetensors"))[0])
        embeddings.weight = torch.nn.Parameter(weights["decoder.lm_head.weight"])
        model.decoder.lm_head.weight = embeddings.weight

    fixed_dir = os.path.join(work_dir, name + "-fixed")
    model.save_pretrained(fixed_dir)
    processor = DonutProcessor.from_pretrained(source)
    processor.save_pretrained(fixed_dir)

    onnx_dir = os.path.join(work_dir, name + "-onnx")
    optimum_cli = os.path.join(os.path.dirname(sys.executable), "optimum-cli")
    subprocess.run([optimum_cli, "export", "onnx", "--model", fixed_dir, "--task", "image-to-text-with-past", onnx_dir], check=True)

    target = os.path.join(output_dir, name)
    os.makedirs(target, exist_ok=True)

    quantize_dynamic(os.path.join(onnx_dir, "encoder_model.onnx"), os.path.join(target, "encoder.onnx"), weight_type=QuantType.QInt8)

    gather_quantized = os.path.join(work_dir, name + "-decoder-gather.onnx")
    quantize_dynamic(os.path.join(onnx_dir, "decoder_model_merged.onnx"), gather_quantized, weight_type=QuantType.QInt8,
                     op_types_to_quantize=["Gather"], extra_options={"EnableSubgraph": True})
    config = DefaultWeightOnlyQuantConfig(block_size=128, is_symmetric=True, bits=8, accuracy_level=4, op_types_to_quantize=("MatMul",))
    quantizer = MatMulNBitsQuantizer(gather_quantized, bits=8, block_size=128, is_symmetric=True, accuracy_level=4,
                                     op_types_to_quantize=("MatMul",), algo_config=config)
    quantizer.process()
    quantizer.model.save_model_to_file(os.path.join(target, "decoder.onnx"), use_external_data_format=False)

    shutil.copy(os.path.join(fixed_dir, "tokenizer.json"), os.path.join(target, "tokenizer.json"))

    size = processor.image_processor.size
    decoder_config = model.config.decoder
    donut_config = {
        "name": name,
        "source": repo,
        "task_prompt": task_prompt,
        "image_height": size["height"],
        "image_width": size["width"],
        "max_length": MAX_LENGTH,
        "decoder_layers": decoder_config.decoder_layers,
        "decoder_attention_heads": decoder_config.decoder_attention_heads,
        "decoder_head_dim": decoder_config.d_model // decoder_config.decoder_attention_heads,
        "encoder_file": "encoder.onnx",
        "decoder_file": "decoder.onnx",
        "tokenizer_file": "tokenizer.json",
    }

    with open(os.path.join(target, "donut.json"), "w") as file:
        json.dump(donut_config, file, indent=2)


def main():
    output_dir = sys.argv[1]

    with tempfile.TemporaryDirectory() as work_dir:
        for name, repo, revision, task_prompt in MODELS:
            convert(name, repo, revision, task_prompt, output_dir, work_dir)


if __name__ == "__main__":
    main()
