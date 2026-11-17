// 用真实的 Response/ReadableStream 控制网络输入，让测试经过 SSE 解析器。
export class SSERequest {
  readonly response: Promise<Response>;
  readonly signal: AbortSignal;
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

  open(): void {
    const body = new ReadableStream<Uint8Array>({
      start: controller => {
        this.controller = controller;
      },
      cancel: () => {
        this.finished = true;
      },
    });
    this.resolve(new Response(body, { headers: { 'Content-Type': 'text/event-stream' } }));
  }

  send(event: string, payload: unknown): void {
    this.controller!.enqueue(
      new TextEncoder().encode(`event: ${event}\ndata: ${JSON.stringify(payload)}\n\n`)
    );
  }

  fail(error = new Error('Connection lost')): void {
    if (this.finished) return;
    this.finished = true;
    if (this.controller) this.controller.error(error);
    else this.reject(error);
  }
}
