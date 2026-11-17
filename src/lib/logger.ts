import { isEnableDebug } from './settings.ts';

function logDebug(msg: string, obj?: unknown): void {
    if (isEnableDebug()) {
        if (obj) {
            console.debug('[ezrbookkeeping Debug] ' + msg, obj);
        } else {
            console.debug('[ezrbookkeeping Debug] ' + msg);
        }
    }
}

function logInfo(msg: string, obj?: unknown): void {
    if (obj) {
        console.info('[ezrbookkeeping Info] ' + msg, obj);
    } else {
        console.info('[ezrbookkeeping Info] ' + msg);
    }
}

function logWarn(msg: string, obj?: unknown): void {
    if (obj) {
        console.warn('[ezrbookkeeping Warn] ' + msg, obj);
    } else {
        console.warn('[ezrbookkeeping Warn] ' + msg);
    }
}

function logError(msg: string, obj?: unknown): void {
    if (obj) {
        console.error('[ezrbookkeeping Error] ' + msg, obj);
    } else {
        console.error('[ezrbookkeeping Error] ' + msg);
    }
}

export default {
    debug: logDebug,
    info: logInfo,
    warn: logWarn,
    error: logError
};
