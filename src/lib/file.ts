import type { ImportFileTypeAndExtensions } from '@/core/import_transaction.ts';
import { KnownFileType } from '@/core/file.ts';

import { isString } from './common.ts';

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

export function isPdfFileUrl(url: string | null | undefined): boolean {
    if (!url || !isString(url)) {
        return false;
    }

    let path = url;
    const queryOrHashIndex = path.search(/[?#]/);

    if (queryOrHashIndex >= 0) {
        path = path.substring(0, queryOrHashIndex);
    }

    const lastSlashIndex = path.lastIndexOf('/');
    const fileName = lastSlashIndex >= 0 ? path.substring(lastSlashIndex + 1) : path;

    if (fileName.indexOf('.') < 0) {
        return false;
    }

    return getFileExtension(fileName).toLowerCase() === KnownFileType.PDF.extension;
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
