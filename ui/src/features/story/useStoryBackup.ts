import { computed, ref } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2StoryBackupDownload,
  getSdApiV2StoryBackupListOptions,
  postSdApiV2StoryBackupBatchDelete,
  type StoryLogBackup,
} from '@/api';
import { downloadApiFile } from '@/api/download';
import { hasAccessToken } from '@/features/auth/state';
import { storyBackupListQueryKey } from './queryKeys';

export function useStoryBackup() {
  const queryClient = useQueryClient();
  const selectedBackups = ref<StoryLogBackup[]>([]);
  const checkAllBackups = ref(false);

  const backupListQuery = useQuery({
    ...getSdApiV2StoryBackupListOptions(),
    enabled: hasAccessToken,
  });

  const backups = computed(() => backupListQuery.data.value?.item.data ?? []);
  const selectedBackupNames = computed({
    get: () => selectedBackups.value.map(item => item.name),
    set: names => {
      selectedBackups.value = backups.value.filter(item => names.includes(item.name));
    },
  });
  const isIndeterminate = computed(() => {
    return selectedBackups.value.length > 0 && selectedBackups.value.length < backups.value.length;
  });
  const selectedBytes = computed(() =>
    selectedBackups.value.map(item => item.fileSize).reduce((sum, size) => sum + size, 0),
  );

  const refreshList = () =>
    queryClient.invalidateQueries({
      queryKey: storyBackupListQueryKey(),
    });

  const deleteMutation = useMutation({
    mutationFn: async (names: string[]) => {
      const { data } = await postSdApiV2StoryBackupBatchDelete({
        body: {
          names,
        },
        throwOnError: true,
      });
      return data.item;
    },
    onSuccess: async () => {
      selectedBackups.value = [];
      checkAllBackups.value = false;
      await refreshList();
    },
  });

  function handleCheckAllChange(checked: boolean) {
    selectedBackups.value = checked ? [...backups.value] : [];
  }

  function handleCheckedBackupChange() {
    checkAllBackups.value = selectedBackups.value.length === backups.value.length;
  }

  async function downloadBackup(name: string) {
    await downloadApiFile(
      getSdApiV2StoryBackupDownload({
        query: { name },
        parseAs: 'blob',
        throwOnError: true,
      }),
      name,
    );
  }

  return {
    backupListQuery,
    backups,
    selectedBackups,
    selectedBackupNames,
    checkAllBackups,
    isIndeterminate,
    selectedBytes,
    deleteMutation,
    refreshList,
    handleCheckAllChange,
    handleCheckedBackupChange,
    downloadBackup,
  };
}
