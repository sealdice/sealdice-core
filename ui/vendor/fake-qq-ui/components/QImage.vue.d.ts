import { MessageProps } from '../types/message';
type __VLS_Props = MessageProps & {
    src: string;
    alt?: string;
    isFile?: boolean;
    fileName?: string;
    fileSize?: string;
    maxWidth?: string;
    maxHeight?: string;
    canDownload?: boolean;
};
declare const _default: import('vue').DefineComponent<__VLS_Props, {}, {}, {}, {}, import('vue').ComponentOptionsMixin, import('vue').ComponentOptionsMixin, {}, string, import('vue').PublicProps, Readonly<__VLS_Props> & Readonly<{}>, {
    self: boolean;
    avatar: string;
    tag: string;
    tagColor: import('..').QTagColors | keyof typeof import('..').QTagColors | import('../lib/QTagCustomize').default;
    isBot: boolean;
    alt: string;
    isFile: boolean;
    fileName: string;
    fileSize: string;
    maxWidth: string;
    maxHeight: string;
    canDownload: boolean;
}, {}, {}, {}, string, import('vue').ComponentProvideOptions, false, {}, HTMLElement>;
export default _default;
