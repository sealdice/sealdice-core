/** Format seconds to m'ss" or ss" */
export declare function formatDuration(duration: number): string;
/**
 * Shared voice playback progress controller.
 * Manages progress bar animation, play/pause state, and duration formatting.
 */
export declare function useVoiceProgress(): {
    progressItemsRef: import('vue').ShallowRef<HTMLDivElement | undefined, HTMLDivElement | undefined>;
    processLinePos: import('vue').ShallowRef<number, number>;
    playEnded: import('vue').ShallowRef<boolean, boolean>;
    playPaused: import('vue').ShallowRef<boolean, boolean>;
    formatedDuration: import('vue').ShallowRef<string, string>;
    /** Call when starting playback from beginning */
    onPlaybackStart: (progressItems: HTMLElement[], duration: number) => void;
    /** Call to pause (e.g. audioCtx.suspend / audio.pause) */
    pause: () => void;
    /** Call to resume (e.g. audioCtx.resume / audio.play) */
    resume: () => void;
    /** Call on playback ended */
    reset: () => void;
};
