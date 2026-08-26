<template>
  <el-config-provider :locale="zhTw">
    <router-view />
  </el-config-provider>
</template>

<script setup lang="ts">
import { ElConfigProvider } from 'element-plus'
import zhTw from 'element-plus/es/locale/lang/zh-tw'
</script>

<style>
/* 根節點樣式由 element-overrides.scss 統籌 */
</style>
