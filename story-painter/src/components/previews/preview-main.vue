<template>
	<div class="preview" id="preview" v-show="isShow">
		<div v-if="previewItems.length === 0">
			<div>染色失败，内容为空或无法识别此格式。</div>
			<div>已知支持的格式有: 海豹Log(json)、赵/Dice!原始文件、塔原始文件</div>
			<div>请先清空编辑框，再行复制</div>
		</div>

		<VirtualList
			class="list-dynamic scroll-touch scroller"
			:data-key="'index'"
			:data-sources="previewItems"
			:data-component="Item"
			:estimate-size="20"
			:item-class="''"
		/>

		<!-- <DynamicScroller class="scroller list-dynamic scroll-touch" :items="testItems" :min-item-size="20" :key-field="'index'">
			<template #default="{ item: i, index, active }">
				<DynamicScrollerItem :item="i" :active="active" :size-dependencies="[i.message]" :data-index="index" :data-active="active">
					<div>
						{{  1111  }}
						<span style="color: #aaa" class="_time" v-if="!store.exportOptions.timeHide">{{ timeSolve(i) }}</span>
						<span :style="{ 'color': colorByName(i) }" class="_nickname">{{ nicknameSolve(i) }}</span>
						<span :style="{ 'color': colorByName(i) }" v-html="previewMessageSolve(i)"></span>
					</div>
				</DynamicScrollerItem>
			</template>
		</DynamicScroller> -->
	</div>

</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { useStore } from '~/store';
import { LogItem, packNameId } from '~/logManager/types';
// @ts-ignore
import VirtualList from 'vue3-virtual-scroll-list';
import Item from './preview-main-item.vue'

const preview = ref(null)

const props = defineProps<{
	isShow: boolean,
	previewItems: LogItem[],
}>();

const store = useStore();

onMounted(() => {
	store.previewElement = preview.value as any;
})
</script>
