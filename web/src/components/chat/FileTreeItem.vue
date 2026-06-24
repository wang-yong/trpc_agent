<script setup lang="ts">
import { ref } from 'vue'
import type { FileNode } from '@/api'

const props = defineProps<{
  node: FileNode
}>()

const collapsed = ref(true)

function handleClick() {
  if (props.node.is_dir) {
    collapsed.value = !collapsed.value
  } else {
    // 触发全局预览广播，携带文件的相对路径
    window.dispatchEvent(new CustomEvent('preview-file', { detail: props.node.path }))
  }
}
</script>

<template>
  <div class="tree-item-node" :class="{ 'is-dir': node.is_dir }">
    <div class="node-label-row" @click="handleClick">
      <!-- 文件夹折叠小箭头 -->
      <span v-if="node.is_dir" class="arrow-icon" :class="{ rotated: !collapsed }">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><polyline points="9 18 15 12 9 6"/></svg>
      </span>
      <span v-else class="arrow-placeholder"></span>

      <!-- 专属高颜值文件/文件夹图标 -->
      <span class="node-icon">
        <template v-if="node.is_dir">
          <!-- 黄色文件夹图标 -->
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="#f59e0b" stroke-width="2.3" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
        </template>
        <template v-else>
          <!-- 蓝色文件图标 -->
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="#6b8bf5" stroke-width="2.3" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><polyline points="13 2 13 9 20 9"/></svg>
        </template>
      </span>

      <!-- 名字 -->
      <span class="node-name" :title="node.name">{{ node.name }}</span>
    </div>

    <!-- 递归自渲染子目录列表 -->
    <Transition name="tree-expand">
      <div v-if="node.is_dir && !collapsed" class="node-children">
        <FileTreeItem
          v-for="child in node.children"
          :key="child.path"
          :node="child"
        />
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.tree-item-node {
  display: flex;
  flex-direction: column;
}
.node-label-row {
  display: flex;
  align-items: center;
  padding: 4px 6px;
  border-radius: 5px;
  cursor: pointer;
  user-select: none;
  transition: background .1s;
}
.node-label-row:hover {
  background: var(--hover-color);
}
.arrow-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  color: var(--text-color-3);
  margin-right: 2px;
  transition: transform .15s ease;
}
.arrow-icon.rotated {
  transform: rotate(90deg);
}
.arrow-placeholder {
  width: 16px;
  flex-shrink: 0;
}
.node-icon {
  display: inline-flex;
  align-items: center;
  margin-right: 6px;
  flex-shrink: 0;
}
.node-name {
  font-size: 12px;
  color: var(--text-color-2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-family: inherit;
}
.node-label-row:hover .node-name {
  color: var(--text-color);
}
.node-children {
  padding-left: 12px; /* 树深度递进缩进 */
  border-left: 1px dashed var(--border-color);
  margin-left: 12px;
  display: flex;
  flex-direction: column;
}

/* Tree expand transition */
.tree-expand-enter-active, .tree-expand-leave-active {
  transition: all .15s ease;
  max-height: 500px;
  overflow: hidden;
}
.tree-expand-enter-from, .tree-expand-leave-to {
  max-height: 0;
  opacity: 0;
}
</style>
