<template>
  <el-card class="profile-info" shadow="never">
    <template #header>
      <div class="info-head">
        <span class="card-title">档案信息</span>
        <el-button v-if="!editing" size="small" text type="primary" @click="editing = true">编辑</el-button>
        <template v-else>
          <el-button size="small" text @click="editing = false">取消</el-button>
        </template>
      </div>
    </template>

    <BirthForm v-if="editing" :submitting="submitting" :initial="profile" @submit="$emit('update', $event)" />

    <el-descriptions v-else :column="1" border>
      <el-descriptions-item label="称呼">{{ profile.nickname }}</el-descriptions-item>
      <el-descriptions-item v-if="profile.name" label="姓名">{{ profile.name }}</el-descriptions-item>
      <el-descriptions-item label="性别">{{ profile.gender === 'male' ? '男' : '女' }}</el-descriptions-item>
      <el-descriptions-item label="出生">{{ birthSummary(profile) }}</el-descriptions-item>
      <el-descriptions-item v-if="profile.relation" label="关系">{{ profile.relation }}</el-descriptions-item>
      <el-descriptions-item v-if="profile.note" label="备注">{{ profile.note }}</el-descriptions-item>
      <el-descriptions-item label="版本">v{{ profile.version }}</el-descriptions-item>
    </el-descriptions>
  </el-card>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { birthSummary, type Profile, type ProfileInput } from '@/api/profiles'
import BirthForm from '@/components/profiles/BirthForm.vue'

defineProps<{ profile: Profile; submitting?: boolean }>()
defineEmits<{ update: [ProfileInput] }>()

const editing = ref(false)
</script>

<style scoped>
.profile-info { border-radius: 12px; margin-bottom: 20px; }
.info-head { display: flex; align-items: center; justify-content: space-between; }
.card-title { font-weight: 600; }
</style>