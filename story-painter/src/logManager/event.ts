// 一个简易事件系统，参考了mitt和nanoevents
// https://github.com/developit/mitt
// https://github.com/ai/nanoevents

// An event handler can take an optional event argument
// and should not return a value
export type EventHandler = (...args: any) => void;

export interface EventsMap {
  [event: string]: any;
}

export class Emitter<T extends EventsMap = EventsMap> {
  sender: any;

  constructor(sender: any) {
    this.sender = sender;
  }

  all = new Map<string, Array<EventHandler>>();

  /**
   * Register an event handler for the given type.
   * @param {string|symbol} type Type of event to listen for, or `"*"` for all events
   * @param {Function} handler Function to call in response to given event
   * @return {Function} handler
   */
  on<K extends keyof T>(type: K, handler: T[K]): T[K] {
    const handlers = this.all.get(type as string);
    const added = handlers && handlers.push(handler);
    if (!added) {
      this.all.set(type as string, [handler]);
    }
    return handler;
  }

  /**
   * Remove an event handler for the given type.
   * @param {string|symbol} type Type of event to unregister `handler` from, or `"*"`
   * @param {Function} handler Handler function to remove
   * @memberOf mitt
   */
  off<K extends keyof T>(type: K, handler: T[K]) {
    const handlers = this.all.get(type as string);
    if (handlers) {
      if (handler) {
        handlers.splice(handlers.indexOf(handler) >>> 0, 1);
      } else {
        handlers.length = 0;
      }

    }
  }

  /**
   * Invoke all handlers for the given type.
   *
   * @param {string|symbol} type The event type to invoke
   * @param {Any} [args] Any value, passed to each handler
   * @memberOf mitt
   */
  emit<K extends keyof T>(type: K, ...args: Parameters<T[K]>) {
    (this.all.get(type as string) || []).slice().map((handler) => {
      handler(...args);
    });
  }


  /**
   * remove all the handles
   */
  dispose() {
    this.all.clear();
  }
}
