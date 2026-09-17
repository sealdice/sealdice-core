# Message Segment Compatibility

## Canonical Model

`Message.Segment` is the canonical message body inside the core. Command parsing, reparsing, and future mixed-content features should use segment-backed data.

`Message.Message` remains available for compatibility with existing Go call sites and JS plugins. It is a derived text view and should not be treated as the source of truth inside new core code.

## Soft-Deprecated Compatibility Views

The following fields remain available but are compatibility views:

| API | Compatibility status | Preferred direction |
| --- | --- | --- |
| `msg.message` | Soft-deprecated text view | `msg.segment` |
| `cmdArgs.rawText` | Soft-deprecated text view | parsed command data |
| `cmdArgs.rawArgs` | Soft-deprecated text view | parsed command data |
| `cmdArgs.cleanArgs` | Soft-deprecated text view | existing helper methods first, segment-aware APIs when introduced |

Compatibility is semantic. The project does not guarantee exact whitespace, exact CQ parameter order, or exact formatting for unknown segment text.

## Supported Legacy Behavior

JS plugins may continue to read `msg.message` and the existing `cmdArgs` string fields.

JS plugins may continue to construct `seal.newMessage()` with only `message` filled when passing the message into supported boundary APIs such as `createTempCtx`.

Existing `CmdArgs` helper methods remain available:

- `getArgN`
- `getKwarg`
- `isArgEqual`
- `eatPrefixWith`
- `chopPrefixToArgsWith`
- `getRestArgsFrom`

## New Code Guidance

New core code should consume `Message.Segment` and avoid reparsing compatibility text.

New command parser code should use the segment projection layer rather than constructing CQ strings.

New JS-facing segment-aware APIs should be designed as wrapper methods so the internal parsed-command model can evolve without exposing implementation details.
