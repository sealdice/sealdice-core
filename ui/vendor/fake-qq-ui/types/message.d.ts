import { default as QTagColors } from '../lib/QTagColors';
import { default as QTagCustomize } from '../lib/QTagCustomize';
export interface MessageProps {
    /** Whether the message is sent by the current user */
    self?: boolean;
    /** Display name of the sender */
    name: string;
    /** URL of the sender's avatar image */
    avatar?: string;
    /** Group tag text shown next to the name */
    tag?: string;
    /** Tag color: preset name, enum value, or custom color object */
    tagColor?: QTagColors | keyof typeof QTagColors | QTagCustomize;
    /** Whether the sender is a bot (shows bot badge) */
    isBot?: boolean;
}
export declare const messagePropsDefaults: {
    readonly self: false;
    readonly avatar: undefined;
    readonly tag: undefined;
    readonly tagColor: undefined;
    readonly isBot: false;
};
