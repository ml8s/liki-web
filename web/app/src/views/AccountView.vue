<template>
  <section v-if="auth.user">
    <h1>账户</h1>
    <el-descriptions :column="1" border>
      <el-descriptions-item label="邮箱">{{ auth.user.email }}</el-descriptions-item>
      <el-descriptions-item label="状态">{{ auth.user.status }}</el-descriptions-item>
    </el-descriptions>
  </section>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
const auth = useAuthStore()
</script>
