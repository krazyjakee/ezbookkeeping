import { describe, expect, it } from 'vitest';

import { isPdfFile, isPdfFileUrl, getFileDownloadUrl } from '@/lib/file.ts';

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
