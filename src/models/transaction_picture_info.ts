export class TransactionPicture implements TransactionPictureInfoBasicResponse {
    public pictureId: string;
    public originalUrl: string;
    public originalFileName?: string;

    private constructor(pictureId: string, originalUrl: string, originalFileName?: string) {
        this.pictureId = pictureId;
        this.originalUrl = originalUrl;

        if (originalFileName) {
            this.originalFileName = originalFileName;
        }
    }

    public static of(picture: TransactionPictureInfoBasicResponse): TransactionPicture {
        return new TransactionPicture(picture.pictureId, picture.originalUrl, picture.originalFileName);
    }

    public static ofMulti(pictureResponses: TransactionPictureInfoBasicResponse[]): TransactionPicture[] {
        const pictures: TransactionPicture[] = [];

        for (const pictureResponse of pictureResponses) {
            pictures.push(TransactionPicture.of(pictureResponse));
        }

        return pictures;
    }
}

export interface TransactionPictureUnusedDeleteRequest {
    readonly id: string;
}

export interface TransactionPictureInfoBasicResponse {
    readonly pictureId: string;
    readonly originalUrl: string;
    readonly originalFileName?: string;
}
