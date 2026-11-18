import type { ImportFileTypeAndExtensions } from '@/core/import_transaction.ts';
import { KnownFileType } from '@/core/file.ts';

import {
    SUPPORTED_IMAGE_EXTENSIONS,
    SUPPORTED_PDF_DOCUMENT_EXTENSIONS,
    SUPPORTED_TRANSACTION_PICTURE_EXTENSIONS
} from '@/consts/file.ts';

import { isString } from './common.ts';

export type AttachmentKind = 'image' | 'pdf' | 'document';
export type DocumentAttachmentType = 'pdf' | 'word' | 'text_document' | 'spreadsheet' | 'delimited_text' | 'plain_text' | 'image' | 'other';

export function getFileExtension(filename: string): string {
    if (!filename || !isString(filename)) {
        return '';
    }

    const parts = filename.split('.');
    return parts[parts.length - 1] as string;
}

export function findExtensionByType(items: ImportFileTypeAndExtensions[] | undefined, type: string): string | undefined {
    if (!items || items.length < 1) {
        return undefined;
    }

    for (const item of items) {
        if (item.type === type) {
            return item.extensions;
        }
    }

    return undefined;
}

export function isFileExtensionSupported(filename: string, supportedExtensions: string): boolean {
    if (!supportedExtensions) {
        return false;
    }

    const supportedExtensionsArray = supportedExtensions.split(',');
    const fileExtension = getFileExtension(filename).toLowerCase();

    for (const supportedExtension of supportedExtensionsArray) {
        if (getFileExtension(supportedExtension).toLowerCase() === fileExtension) {
            return true;
        }
    }

    return false;
}

export function isPdfFile(file: File | null | undefined): boolean {
    if (!file) {
        return false;
    }

    if (file.type && KnownFileType.PDF.isSameType(file.type)) {
        return true;
    }

    return getFileExtension(file.name).toLowerCase() === KnownFileType.PDF.extension;
}

export function getFileExtensionFromUrl(url: string | null | undefined): string {
    if (!url || !isString(url)) {
        return '';
    }

    let path = url;
    const queryOrHashIndex = path.search(/[?#]/);

    if (queryOrHashIndex >= 0) {
        path = path.substring(0, queryOrHashIndex);
    }

    const lastSlashIndex = path.lastIndexOf('/');
    const fileName = lastSlashIndex >= 0 ? path.substring(lastSlashIndex + 1) : path;

    if (fileName.indexOf('.') < 0) {
        return '';
    }

    return getFileExtension(fileName).toLowerCase();
}

export function isPdfFileUrl(url: string | null | undefined): boolean {
    return getFileExtensionFromUrl(url) === KnownFileType.PDF.extension;
}

function isExtensionInList(extension: string, supportedExtensions: string): boolean {
    if (!extension) {
        return false;
    }

    for (const supportedExtension of supportedExtensions.split(',')) {
        if (getFileExtension(supportedExtension.trim()).toLowerCase() === extension) {
            return true;
        }
    }

    return false;
}

/**
 * Returns how a transaction attachment with the specified (lowercase, without dot) file extension is displayed:
 * 'image' can be displayed inline, 'pdf' can be displayed by the built-in pdf viewer of browsers,
 * and 'document' (office files, text files, heic / heif images and unknown types) can only be downloaded
 */
export function getAttachmentKindFromExtension(extension: string | null | undefined): AttachmentKind {
    const ext = extension ? extension.toLowerCase() : '';

    if (isExtensionInList(ext, SUPPORTED_IMAGE_EXTENSIONS)) {
        return 'image';
    } else if (isExtensionInList(ext, SUPPORTED_PDF_DOCUMENT_EXTENSIONS)) {
        return 'pdf';
    } else {
        return 'document';
    }
}

export function getAttachmentKindFromUrl(url: string | null | undefined): AttachmentKind {
    return getAttachmentKindFromExtension(getFileExtensionFromUrl(url));
}

export function getAttachmentKindFromFile(file: File | null | undefined): AttachmentKind {
    if (!file) {
        return 'document';
    }

    const extension = getFileExtension(file.name).toLowerCase();

    if (file.name && file.name.indexOf('.') >= 0 && isTransactionAttachmentExtensionSupported(extension)) {
        return getAttachmentKindFromExtension(extension);
    }

    if (file.type) {
        if (KnownFileType.PDF.isSameType(file.type)) {
            return 'pdf';
        }

        for (const imageExtension of SUPPORTED_IMAGE_EXTENSIONS.split(',')) {
            const fileType = KnownFileType.parse(getFileExtension(imageExtension.trim()).toLowerCase());

            if (fileType && fileType.isSameType(file.type)) {
                return 'image';
            }
        }
    }

    return 'document';
}

export function getDocumentAttachmentType(extension: string | null | undefined): DocumentAttachmentType {
    switch (extension ? extension.toLowerCase() : '') {
        case KnownFileType.PDF.extension:
            return 'pdf';
        case KnownFileType.DOCX.extension:
            return 'word';
        case KnownFileType.ODT.extension:
            return 'text_document';
        case KnownFileType.XLSX.extension:
        case KnownFileType.ODS.extension:
            return 'spreadsheet';
        case KnownFileType.CSV.extension:
            return 'delimited_text';
        case KnownFileType.TXT.extension:
            return 'plain_text';
        case KnownFileType.HEIC.extension:
        case KnownFileType.HEIF.extension:
            return 'image';
        default:
            return 'other';
    }
}

/**
 * Returns the display name of a transaction attachment, which is the original file name when it is known,
 * otherwise a localized "PDF Document" for pdf documents or the upper case file extension (e.g. "DOCX") for other documents
 */
export function getTransactionAttachmentDisplayName(originalFileName: string | null | undefined, url: string | null | undefined, tt: (key: string) => string): string {
    if (originalFileName) {
        return originalFileName;
    }

    const extension = getFileExtensionFromUrl(url);

    if (getAttachmentKindFromExtension(extension) === 'pdf') {
        return tt('PDF Document');
    }

    return extension ? extension.toUpperCase() : tt('Document');
}

export function isTransactionAttachmentExtensionSupported(extension: string | null | undefined): boolean {
    return isExtensionInList(extension ? extension.toLowerCase() : '', SUPPORTED_TRANSACTION_PICTURE_EXTENSIONS);
}

export function isSupportedTransactionAttachmentFile(file: File | null | undefined): boolean {
    if (!file) {
        return false;
    }

    if (file.name && file.name.indexOf('.') >= 0 && isTransactionAttachmentExtensionSupported(getFileExtension(file.name))) {
        return true;
    }

    // files without supported file extension (e.g. pasted or captured files) are accepted when their content type is a supported image or pdf
    return getAttachmentKindFromFile(file) !== 'document';
}

export function getFileDownloadUrl(url: string | null | undefined): string | undefined {
    if (!url || !isString(url)) {
        return undefined;
    }

    const hashIndex = url.indexOf('#');
    const urlWithoutHash = hashIndex >= 0 ? url.substring(0, hashIndex) : url;
    const hash = hashIndex >= 0 ? url.substring(hashIndex) : '';
    const separator = urlWithoutHash.indexOf('?') >= 0 ? '&' : '?';

    return `${urlWithoutHash}${separator}download=true${hash}`;
}
