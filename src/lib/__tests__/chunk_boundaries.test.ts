import { describe, expect, it } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';

// vite.config.ts puts these source paths into the shared "common" chunk, which both desktop.html and mobile.html load at startup
const COMMON_CHUNK_PATHS: string[] = [
	'core',
	'consts',
	'models',
	'stores',
	'lib/map',
	'lib/calendar',
	'lib/ui/common.ts',
	'components/base',
	'components/common',
	'views/base',
	'locales/helpers.ts',
	'locales/index.ts'
];

// packages the build groups into "vendor-desktop" (Vuetify, vue-echarts and therefore ECharts) or "vendor-mobile" (Framework7)
const PLATFORM_ONLY_IMPORTS: RegExp = /^(vuetify|vue-router|vue3-perfect-scrollbar|perfect-scrollbar|vuedraggable|sortablejs|@mdi\/|vue-echarts|framework7|dom7|swiper|skeleton-elements)|^@\/(lib\/ui\/(desktop|mobile)|components\/(desktop|mobile)|views\/(desktop|mobile))(\/|\.|$)/;

const SRC_DIR: string = path.resolve(__dirname, '../..');

function listSourceFiles(relativePath: string): string[] {
	const fullPath = path.join(SRC_DIR, relativePath);

	if (!fs.existsSync(fullPath)) {
		return [];
	}

	if (fs.statSync(fullPath).isFile()) {
		return [fullPath];
	}

	const files: string[] = [];

	for (const entry of fs.readdirSync(fullPath, { withFileTypes: true })) {
		if (entry.isDirectory()) {
			if (entry.name !== '__tests__') {
				files.push(...listSourceFiles(path.join(relativePath, entry.name)));
			}
		} else if (/\.(ts|vue)$/.test(entry.name)) {
			files.push(path.join(fullPath, entry.name));
		}
	}

	return files;
}

function getStaticRuntimeImports(content: string): string[] {
	const imports: string[] = [];

	for (const match of content.matchAll(/^\s*import\s+(?!type\b)([^'";]*?\s+from\s+)?['"]([^'"]+)['"]/gm)) {
		imports.push(match[2] as string);
	}

	return imports;
}

describe('shared chunk boundaries', () => {
	// lib/*.ts files directly under src/lib are also in the common chunk
	const files = [
		...COMMON_CHUNK_PATHS.flatMap(p => listSourceFiles(p)),
		...fs.readdirSync(path.join(SRC_DIR, 'lib')).filter(name => /\.ts$/.test(name)).map(name => path.join(SRC_DIR, 'lib', name))
	];

	it('should find the shared source files', () => {
		expect(files.length).toBeGreaterThan(50);
	});

	it('should not statically import desktop-only or mobile-only modules from shared code', () => {
		const violations: string[] = [];

		for (const file of files) {
			for (const imported of getStaticRuntimeImports(fs.readFileSync(file, 'utf-8'))) {
				if (PLATFORM_ONLY_IMPORTS.test(imported)) {
					violations.push(`${path.relative(SRC_DIR, file)} imports ${imported}`);
				}
			}
		}

		expect(violations).toEqual([]);
	});
});
