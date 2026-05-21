import { onBeforeUnmount, shallowRef, watch, type MaybeRefOrGetter, toValue } from 'vue';
import { getSdApiV2ResourceData, type ResourceItem } from '@/api';

export function useResourcePreview(
  item: MaybeRefOrGetter<ResourceItem>,
  thumbnail: MaybeRefOrGetter<boolean>,
) {
  const objectUrl = shallowRef('');
  const loading = shallowRef(false);
  const failed = shallowRef(false);
  let loadId = 0;

  function revokeObjectUrl() {
    if (!objectUrl.value) return;
    URL.revokeObjectURL(objectUrl.value);
    objectUrl.value = '';
  }

  async function loadImage() {
    const currentId = ++loadId;
    const currentItem = toValue(item);
    revokeObjectUrl();
    failed.value = false;

    if (currentItem.type !== 'image' || !currentItem.path) {
      failed.value = true;
      return;
    }

    loading.value = true;
    try {
      const { data } = await getSdApiV2ResourceData({
        query: {
          path: currentItem.path,
          thumbnail: toValue(thumbnail),
        },
        parseAs: 'blob',
        throwOnError: true,
      });
      if (currentId !== loadId) return;
      if (!(data instanceof Blob)) {
        failed.value = true;
        return;
      }
      objectUrl.value = URL.createObjectURL(data);
    } catch {
      if (currentId === loadId) {
        failed.value = true;
      }
    } finally {
      if (currentId === loadId) {
        loading.value = false;
      }
    }
  }

  watch(
    () => [toValue(item).path, toValue(thumbnail)] as const,
    () => {
      void loadImage();
    },
    { immediate: true },
  );

  onBeforeUnmount(() => {
    loadId += 1;
    revokeObjectUrl();
  });

  return {
    objectUrl,
    loading,
    failed,
  };
}
