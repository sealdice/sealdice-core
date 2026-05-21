import { MessageProps } from '../types/message';
type __VLS_Props = MessageProps & {
    fileName: string;
    fileSize?: string;
    fileSrc?: string;
    iconSrc?: string;
    canDownload?: boolean;
};
declare const _default: import('vue').DefineComponent<__VLS_Props, {}, {}, {}, {}, import('vue').ComponentOptionsMixin, import('vue').ComponentOptionsMixin, {}, string, import('vue').PublicProps, Readonly<__VLS_Props> & Readonly<{}>, {
    self: boolean;
    avatar: string;
    tag: string;
    tagColor: import('..').QTagColors | keyof typeof import('..').QTagColors | import('../lib/QTagCustomize').default;
    isBot: boolean;
    fileSize: string;
    canDownload: boolean;
    fileSrc: string;
    iconSrc: string;
}, {}, {}, {}, string, import('vue').ComponentProvideOptions, false, {}, HTMLElement>;
export default _default;
