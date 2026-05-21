import { default as QTagColors } from '../../lib/QTagColors';
import { default as QTagCustomize } from '../../lib/QTagCustomize';
type __VLS_Props = {
    self?: boolean;
    name: string;
    avatar?: string;
    tag?: string;
    tagColor?: QTagColors | keyof typeof QTagColors | QTagCustomize;
    isBot?: boolean;
    src: string;
    text?: string;
    play: () => void;
    playPaused: boolean;
    formatedDuration: string;
};
declare function __VLS_template(): {
    attrs: Partial<{}>;
    slots: {
        default?(_: {}): any;
    };
    refs: {};
    rootEl: HTMLElement;
};
type __VLS_TemplateResult = ReturnType<typeof __VLS_template>;
declare const __VLS_component: import('vue').DefineComponent<__VLS_Props, {}, {}, {}, {}, import('vue').ComponentOptionsMixin, import('vue').ComponentOptionsMixin, {}, string, import('vue').PublicProps, Readonly<__VLS_Props> & Readonly<{}>, {
    text: string;
    self: boolean;
    avatar: string;
    tag: string;
    tagColor: QTagColors | keyof typeof QTagColors | QTagCustomize;
    isBot: boolean;
}, {}, {}, {}, string, import('vue').ComponentProvideOptions, false, {}, HTMLElement>;
declare const _default: __VLS_WithTemplateSlots<typeof __VLS_component, __VLS_TemplateResult["slots"]>;
export default _default;
type __VLS_WithTemplateSlots<T, S> = T & {
    new (): {
        $slots: S;
    };
};
