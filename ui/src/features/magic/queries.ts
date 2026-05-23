import { useMutation } from '@tanstack/vue-query';
import { postSdApiV2MagicSourceInspect } from '@/api';

export interface MagicInspectForm {
  kind: 'sqlite' | 'mysql' | 'postgres';
  sqlitePath: string;
  dsn: string;
}

export function useMagicInspectMutation() {
  return useMutation({
    mutationFn: async (form: MagicInspectForm) => {
      const { data } = await postSdApiV2MagicSourceInspect({
        body: {
          source: {
            kind: form.kind,
            sqlitePath: form.kind === 'sqlite' ? form.sqlitePath.trim() : undefined,
            dsn: form.kind === 'sqlite' ? undefined : form.dsn.trim(),
          },
        },
        throwOnError: true,
      });
      return data.item;
    },
  });
}
