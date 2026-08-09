export interface PprofEntry {
  key: string;
  title: string;
  desc: string;
  binaryPath: string;
  textPath: string;
  filename: string;
  secondsModel?: 'profile' | 'trace';
}

const PPROF_ENTRIES: PprofEntry[] = [
  {
    key: 'profile',
    title: 'CPU Profile',
    desc: '在指定时长内采集 CPU 使用情况。',
    binaryPath: '',
    textPath: '',
    filename: 'cpu.pprof',
    secondsModel: 'profile',
  },
  {
    key: 'trace',
    title: '执行轨迹 (Trace)',
    desc: '在指定时长内采集程序执行轨迹。',
    binaryPath: '',
    textPath: '',
    filename: 'trace.out',
    secondsModel: 'trace',
  },
  {
    key: 'heap',
    title: '堆内存 (Heap)',
    desc: '当前堆内存分配采样，可用于分析内存占用与泄漏。',
    binaryPath: 'heap',
    textPath: 'heap',
    filename: 'heap.pprof',
  },
  {
    key: 'allocs',
    title: '内存分配 (Allocs)',
    desc: '所有已分配的内存采样（历史累计），可用于分析分配热点。',
    binaryPath: 'allocs',
    textPath: 'allocs',
    filename: 'allocs.pprof',
  },
  {
    key: 'goroutine',
    title: '协程 (Goroutine)',
    desc: '当前所有 goroutine 的栈信息，常用于排查死锁、协程泄漏。',
    binaryPath: 'goroutine',
    textPath: 'goroutine',
    filename: 'goroutine.pprof',
  },
  {
    key: 'block',
    title: '阻塞 (Block)',
    desc: '同步原语上的阻塞事件采样。',
    binaryPath: 'block',
    textPath: 'block',
    filename: 'block.pprof',
  },
  {
    key: 'mutex',
    title: '互斥锁 (Mutex)',
    desc: '互斥锁争用事件采样。',
    binaryPath: 'mutex',
    textPath: 'mutex',
    filename: 'mutex.pprof',
  },
  {
    key: 'threadcreate',
    title: '线程创建 (ThreadCreate)',
    desc: '系统线程创建事件的采样。',
    binaryPath: 'threadcreate',
    textPath: 'threadcreate',
    filename: 'threadcreate.pprof',
  },
  {
    key: 'cmdline',
    title: '命令行参数',
    desc: '显示进程启动的命令行参数。',
    binaryPath: 'cmdline',
    textPath: 'cmdline',
    filename: 'cmdline.txt',
  },
];

export function createPprofEntries(): PprofEntry[] {
  return PPROF_ENTRIES;
}

export function buildPprofBinaryPath(
  entry: PprofEntry,
  options: { profileSeconds: number; traceSeconds: number },
): string {
  if (entry.key === 'profile') return `/profile?seconds=${options.profileSeconds}`;
  if (entry.key === 'trace') return `/trace?seconds=${options.traceSeconds}`;
  return `/${entry.binaryPath}?debug=0`;
}

export function buildPprofTextPath(entry: PprofEntry): string {
  if (!entry.textPath) return '';
  return `/${entry.textPath}?debug=1`;
}
