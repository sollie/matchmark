import { settle, type Settled } from "./api/util";

type Call<T> = Promise<{ data?: T; error?: unknown; response: Response }>;

export class Command {
  error = $state("");
  notice = $state("");
  readonly #reload: () => Promise<void>;

  constructor(reload: () => Promise<void>) {
    this.#reload = reload;
  }

  clear(): void {
    this.error = "";
    this.notice = "";
  }

  fail(message: string): void {
    this.error = message;
  }

  async run<T>(
    call: Call<T>,
    notice?: string | ((data: T) => string),
    reload: (data: T) => Promise<void> = this.#reload,
  ): Promise<Settled<T>> {
    this.clear();
    const result = await settle(call);
    if (!result.ok) {
      this.error = result.error;
      return result;
    }
    if (notice !== undefined) {
      this.notice = typeof notice === "function" ? notice(result.data) : notice;
    }
    await reload(result.data);
    return result;
  }
}
