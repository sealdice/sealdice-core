import { createDiscreteApi } from 'naive-ui';
import type { HttpErrorFeedback } from './httpStatusFeedback';

const { dialog, message } = createDiscreteApi(['dialog', 'message']);

let activeDialogKey = '';

function showDialog(feedback: Extract<HttpErrorFeedback, { kind: 'dialog' }>, onClearSession: () => void) {
  const key = `${feedback.title}:${feedback.content}`;
  if (activeDialogKey === key) return;

  activeDialogKey = key;
  dialog.warning({
    title: feedback.title,
    content: feedback.content,
    positiveText: feedback.positiveText,
    negativeText: feedback.negativeText,
    maskClosable: false,
    onPositiveClick: () => {
      if (feedback.clearSession) {
        onClearSession();
      }
    },
    onAfterLeave: () => {
      if (activeDialogKey === key) {
        activeDialogKey = '';
      }
    },
  });
}

export function showApiFeedback(feedback: HttpErrorFeedback, onClearSession: () => void): void {
  if (feedback.kind === 'dialog') {
    showDialog(feedback, onClearSession);
    return;
  }

  message.error(feedback.content, {
    closable: true,
    duration: 5000,
  });
}
