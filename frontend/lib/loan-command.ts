import type { Command, Receipt } from "./loans";
type Storage = Pick<globalThis.Storage, "setItem" | "removeItem">;
// Persist before sending: an interrupted response must replay the exact same command.
export async function submitLoanCommand(command: Command, storage: Storage, storageKey: string, send: (command: Command) => Promise<Receipt>): Promise<Receipt> {
    storage.setItem(storageKey, JSON.stringify(command));
    try {
        const result = await send(command);
        storage.removeItem(storageKey);
        return result;
    }
    catch (error) {
        const status = error instanceof Error && "status" in error ? error.status : undefined;
        if (typeof status === "number" && status >= 400 && status < 500 && status !== 401 && status !== 403 && status !== 408 && status !== 429)
            storage.removeItem(storageKey);
        throw error;
    }
}
