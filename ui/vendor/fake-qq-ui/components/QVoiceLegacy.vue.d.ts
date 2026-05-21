import { MessageProps } from '../types/message';
type __VLS_Props = MessageProps & {
    src: string;
    text?: string;
    volume?: number;
};
declare const _default: import('vue').DefineComponent<__VLS_Props, {}, {}, {}, {}, import('vue').ComponentOptionsMixin, import('vue').ComponentOptionsMixin, {}, string, import('vue').PublicProps, Readonly<__VLS_Props> & Readonly<{}>, {
    text: string;
    self: boolean;
    avatar: string;
    tag: string;
    tagColor: import('..').QTagColors | keyof typeof import('..').QTagColors | import('../lib/QTagCustomize').default;
    isBot: boolean;
    volume: number;
}, {}, {}, {}, string, import('vue').ComponentProvideOptions, false, {
    audio: HTMLAudioElement;
    progressItemsRef: HTMLDivElement;
}, HTMLElement>;
export default _default;
