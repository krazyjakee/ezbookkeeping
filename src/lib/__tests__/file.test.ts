import { describe, expect, it } from 'vitest';

import {
    isPdfFile,
    isPdfFileUrl,
    getFileDownloadUrl,
    getFileExtensionFromUrl,
    getAttachmentKindFromExtension,
    getAttachmentKindFromUrl,
    getAttachmentKindFromFile,
    getDocumentAttachmentType,
    getTransactionAttachmentDisplayName,
    isTransactionAttachmentExtensionSupported,
    isSupportedTransactionAttachmentFile
} from '@/lib/file.ts';
import { SUPPORTED_TRANSACTION_PICTURE_EXTENSIONS } from '@/consts/file.ts';

describe('isPdfFile', () => {
    it('should return false for null or undefined', () => {
        expect(isPdfFile(null)).toBe(false);
        expect(isPdfFile(undefined)).toBe(false);
    });

    it('should return true for pdf content type', () => {
        expect(isPdfFile(new File(['%PDF-1.7'], 'document', { type: 'application/pdf' }))).toBe(true);
    });

    it('should return true for pdf file extension without content type', () => {
        expect(isPdfFile(new File(['%PDF-1.7'], 'invoice.PDF'))).toBe(true);
    });

    it('should return false for image files', () => {
        expect(isPdfFile(new File(['data'], 'image.jpg', { type: 'image/jpeg' }))).toBe(false);
        expect(isPdfFile(new File(['data'], 'image.png'))).toBe(false);
    });
});

describe('isPdfFileUrl', () => {
    it('should return false for empty values', () => {
        expect(isPdfFileUrl(null)).toBe(false);
        expect(isPdfFileUrl(undefined)).toBe(false);
        expect(isPdfFileUrl('')).toBe(false);
    });

    it('should detect pdf transaction picture urls', () => {
        expect(isPdfFileUrl('https://foo.bar/pictures/123.pdf')).toBe(true);
        expect(isPdfFileUrl('https://foo.bar/pictures/123.pdf?token=abc.def')).toBe(true);
        expect(isPdfFileUrl('https://foo.bar/pictures/123.PDF#page=2')).toBe(true);
    });

    it('should return false for image transaction picture urls', () => {
        expect(isPdfFileUrl('https://foo.bar/pictures/123.jpg')).toBe(false);
        expect(isPdfFileUrl('https://foo.bar/pictures/123.png?token=abc.pdf')).toBe(false);
    });

    it('should return false for urls without file extension', () => {
        expect(isPdfFileUrl('https://foo.bar/pdf')).toBe(false);
        expect(isPdfFileUrl('https://foo.bar/pictures/')).toBe(false);
    });
});

describe('getFileDownloadUrl', () => {
    it('should return undefined for empty values', () => {
        expect(getFileDownloadUrl(null)).toBeUndefined();
        expect(getFileDownloadUrl(undefined)).toBeUndefined();
        expect(getFileDownloadUrl('')).toBeUndefined();
    });

    it('should append download parameter to url without query string', () => {
        expect(getFileDownloadUrl('https://foo.bar/pictures/123.pdf')).toBe('https://foo.bar/pictures/123.pdf?download=true');
    });

    it('should append download parameter to url with query string', () => {
        expect(getFileDownloadUrl('https://foo.bar/pictures/123.pdf?token=abc')).toBe('https://foo.bar/pictures/123.pdf?token=abc&download=true');
    });

    it('should keep the hash at the end of url', () => {
        expect(getFileDownloadUrl('https://foo.bar/pictures/123.pdf?token=abc#page=2')).toBe('https://foo.bar/pictures/123.pdf?token=abc&download=true#page=2');
    });
});

describe('getFileExtensionFromUrl', () => {
    it('should return empty string for empty values or urls without file extension', () => {
        expect(getFileExtensionFromUrl(null)).toBe('');
        expect(getFileExtensionFromUrl(undefined)).toBe('');
        expect(getFileExtensionFromUrl('')).toBe('');
        expect(getFileExtensionFromUrl('https://foo.bar/pictures/123')).toBe('');
        expect(getFileExtensionFromUrl('https://foo.bar/pictures/')).toBe('');
    });

    it('should return the lower case file extension and ignore query string and hash', () => {
        expect(getFileExtensionFromUrl('https://foo.bar/pictures/123.DOCX?token=abc.pdf')).toBe('docx');
        expect(getFileExtensionFromUrl('https://foo.bar/pictures/123.csv#abc.png')).toBe('csv');
        expect(getFileExtensionFromUrl('/pictures/123.heic')).toBe('heic');
    });
});

describe('getAttachmentKindFromExtension', () => {
    it('should return image for previewable images', () => {
        for (const ext of ['jpg', 'jpeg', 'png', 'gif', 'webp', 'JPG']) {
            expect(getAttachmentKindFromExtension(ext)).toBe('image');
        }
    });

    it('should return pdf for pdf documents', () => {
        expect(getAttachmentKindFromExtension('pdf')).toBe('pdf');
        expect(getAttachmentKindFromExtension('PDF')).toBe('pdf');
    });

    it('should return document for download-only documents and unknown extensions', () => {
        for (const ext of ['docx', 'xlsx', 'odt', 'ods', 'txt', 'csv', 'heic', 'heif', 'exe', '', null, undefined]) {
            expect(getAttachmentKindFromExtension(ext)).toBe('document');
        }
    });
});

describe('getAttachmentKindFromUrl', () => {
    it('should detect the attachment kind of transaction picture urls', () => {
        expect(getAttachmentKindFromUrl('https://foo.bar/pictures/123.jpg?token=abc')).toBe('image');
        expect(getAttachmentKindFromUrl('https://foo.bar/pictures/123.webp')).toBe('image');
        expect(getAttachmentKindFromUrl('https://foo.bar/pictures/123.pdf?token=abc.jpg')).toBe('pdf');
        expect(getAttachmentKindFromUrl('https://foo.bar/pictures/123.docx?token=abc')).toBe('document');
        expect(getAttachmentKindFromUrl('https://foo.bar/pictures/123.xlsx')).toBe('document');
        expect(getAttachmentKindFromUrl('https://foo.bar/pictures/123.csv')).toBe('document');
        expect(getAttachmentKindFromUrl('https://foo.bar/pictures/123.HEIC')).toBe('document');
    });

    it('should return document for empty urls or urls without file extension', () => {
        expect(getAttachmentKindFromUrl(undefined)).toBe('document');
        expect(getAttachmentKindFromUrl('https://foo.bar/pictures/123')).toBe('document');
    });
});

describe('getAttachmentKindFromFile', () => {
    it('should detect the attachment kind by file extension', () => {
        expect(getAttachmentKindFromFile(new File(['data'], 'receipt.JPEG'))).toBe('image');
        expect(getAttachmentKindFromFile(new File(['%PDF-1.7'], 'invoice.pdf'))).toBe('pdf');
        expect(getAttachmentKindFromFile(new File(['PK'], 'statement.xlsx'))).toBe('document');
        expect(getAttachmentKindFromFile(new File(['data'], 'photo.heic', { type: 'image/heic' }))).toBe('document');
    });

    it('should detect the attachment kind by content type when the file has no supported extension', () => {
        expect(getAttachmentKindFromFile(new File(['data'], 'image', { type: 'image/png' }))).toBe('image');
        expect(getAttachmentKindFromFile(new File(['%PDF-1.7'], 'document', { type: 'application/pdf' }))).toBe('pdf');
        expect(getAttachmentKindFromFile(new File(['data'], 'file', { type: 'application/octet-stream' }))).toBe('document');
    });

    it('should return document for null or undefined', () => {
        expect(getAttachmentKindFromFile(null)).toBe('document');
        expect(getAttachmentKindFromFile(undefined)).toBe('document');
    });
});

describe('getDocumentAttachmentType', () => {
    it('should return the document type of file extensions', () => {
        expect(getDocumentAttachmentType('pdf')).toBe('pdf');
        expect(getDocumentAttachmentType('DOCX')).toBe('word');
        expect(getDocumentAttachmentType('odt')).toBe('text_document');
        expect(getDocumentAttachmentType('xlsx')).toBe('spreadsheet');
        expect(getDocumentAttachmentType('ods')).toBe('spreadsheet');
        expect(getDocumentAttachmentType('csv')).toBe('delimited_text');
        expect(getDocumentAttachmentType('txt')).toBe('plain_text');
        expect(getDocumentAttachmentType('heic')).toBe('image');
        expect(getDocumentAttachmentType('heif')).toBe('image');
        expect(getDocumentAttachmentType('zip')).toBe('other');
        expect(getDocumentAttachmentType(undefined)).toBe('other');
    });
});

describe('getTransactionAttachmentDisplayName', () => {
    const tt = (key: string): string => `[${key}]`;

    it('should return the original file name when it is known', () => {
        expect(getTransactionAttachmentDisplayName('invoice.docx', 'https://foo.bar/pictures/123.docx', tt)).toBe('invoice.docx');
    });

    it('should return the localized pdf document name for pdf documents without original file name', () => {
        expect(getTransactionAttachmentDisplayName(undefined, 'https://foo.bar/pictures/123.pdf?token=abc', tt)).toBe('[PDF Document]');
    });

    it('should return the upper case file extension for other documents without original file name', () => {
        expect(getTransactionAttachmentDisplayName('', 'https://foo.bar/pictures/123.xlsx?token=abc', tt)).toBe('XLSX');
        expect(getTransactionAttachmentDisplayName(null, 'https://foo.bar/pictures/123', tt)).toBe('[Document]');
    });
});

describe('isTransactionAttachmentExtensionSupported', () => {
    it('should accept all supported transaction attachment extensions', () => {
        for (const ext of ['jpg', 'jpeg', 'png', 'gif', 'webp', 'pdf', 'docx', 'xlsx', 'odt', 'ods', 'txt', 'csv', 'heic', 'heif', 'DOCX']) {
            expect(isTransactionAttachmentExtensionSupported(ext)).toBe(true);
        }
    });

    it('should reject unsupported extensions', () => {
        for (const ext of ['doc', 'xls', 'zip', 'exe', 'svg', '', null, undefined]) {
            expect(isTransactionAttachmentExtensionSupported(ext)).toBe(false);
        }
    });

    it('should match the file input accept list', () => {
        expect(SUPPORTED_TRANSACTION_PICTURE_EXTENSIONS.split(',')).toEqual(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.pdf', '.docx', '.xlsx', '.odt', '.ods', '.txt', '.csv', '.heic', '.heif']);
    });
});

describe('isSupportedTransactionAttachmentFile', () => {
    it('should accept files with supported extensions', () => {
        expect(isSupportedTransactionAttachmentFile(new File(['data'], 'receipt.png'))).toBe(true);
        expect(isSupportedTransactionAttachmentFile(new File(['PK'], 'invoice.docx'))).toBe(true);
        expect(isSupportedTransactionAttachmentFile(new File(['a,b'], 'statement.CSV'))).toBe(true);
        expect(isSupportedTransactionAttachmentFile(new File(['data'], 'photo.heif'))).toBe(true);
    });

    it('should accept files without extension when the content type is a supported image or pdf', () => {
        expect(isSupportedTransactionAttachmentFile(new File(['data'], 'image', { type: 'image/jpeg' }))).toBe(true);
        expect(isSupportedTransactionAttachmentFile(new File(['%PDF-1.7'], 'document', { type: 'application/pdf' }))).toBe(true);
    });

    it('should reject unsupported files', () => {
        expect(isSupportedTransactionAttachmentFile(null)).toBe(false);
        expect(isSupportedTransactionAttachmentFile(new File(['data'], 'archive.zip', { type: 'application/zip' }))).toBe(false);
        expect(isSupportedTransactionAttachmentFile(new File(['data'], 'legacy.doc', { type: 'application/msword' }))).toBe(false);
        expect(isSupportedTransactionAttachmentFile(new File(['data'], 'image.bmp', { type: 'image/bmp' }))).toBe(false);
    });
});
