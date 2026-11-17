// 用真实的 Response/ReadableStream 控制网络输入，让测试经过 SSE 解析器。
export class SSERequest {
  readonly response: Promise<Response>;
  readonly signal: AbortSignal;
  cancelled = false;
  private controller?: ReadableStreamDefaultController<Uint8Array>;
  private resolve!: (response: Response) => void;
  private reject!: (error: Error) => void;
  private finished = false;

  constructor(
    readonly url: string,
    readonly options: RequestInit
  ) {
    this.signal = options.signal!;
    this.response = new Promise((resolve, reject) => {
      this.resolve = resolve;
      this.reject = reject;
    });
    this.signal.addEventListener('abort', () =>
      this.fail(new DOMException('Aborted', 'AbortError'))
    );
  }

  open(status = 200, contentType = 'text/event-stream; charset=utf-8'): void {
    const body = new ReadableStream<Uint8Array>({
      start: controller => {
        this.controller = controller;
      },
      cancel: () => {
        this.cancelled = true;
        this.finished = true;
      },
    });
    this.resolve(new Response(body, { status, headers: { 'Content-Type': contentType } }));
  }

  push(chunk: string | Uint8Array): void {
    this.controller!.enqueue(typeof chunk === 'string' ? new TextEncoder().encode(chunk) : chunk);
  }

  send(event: string, payload: unknown): void {
    this.push(`event: ${event}\ndata: ${JSON.stringify(payload)}\n\n`);
  }

  fail(error = new Error('Connection lost')): void {
    if (this.finished) return;
    this.finished = true;
    if (this.controller) this.controller.error(error);
    else this.reject(error);
  }

  end(): void {
    this.finished = true;
    this.controller!.close();
  }
}
