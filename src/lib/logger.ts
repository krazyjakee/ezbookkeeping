import { isEnableDebug } from './settings.ts';

function logDebug(msg: string, obj?: unknown): void {
    if (isEnableDebug()) {
        if (obj) {
            console.debug('[ezbookkeeping2 Debug] ' + msg, obj);
        } else {
            console.debug('[ezbookkeeping2 Debug] ' + msg);
        }
    }
}

function logInfo(msg: string, obj?: unknown): void {
    if (obj) {
        console.info('[ezbookkeeping2 Info] ' + msg, obj);
    } else {
        console.info('[ezbookkeeping2 Info] ' + msg);
    }
}

function logWarn(msg: string, obj?: unknown): void {
    if (obj) {
        console.warn('[ezbookkeeping2 Warn] ' + msg, obj);
    } else {
        console.warn('[ezbookkeeping2 Warn] ' + msg);
    }
}

function logError(msg: string, obj?: unknown): void {
    if (obj) {
        console.error('[ezbookkeeping2 Error] ' + msg, obj);
    } else {
        console.error('[ezbookkeeping2 Error] ' + msg);
    }
}

export default {
    debug: logDebug,
    info: logInfo,
    warn: logWarn,
    error: logError
};
