<template>
  <section>
    <AppPageHeader
      :title="title"
      :subtitle="subtitle"
    >
      <template #actions>
        <el-button
          v-if="showRefresh"
          :loading="loading"
          @click="$emit('refresh')"
        >
          {{ $t('common.refresh') }}
        </el-button>
      </template>
    </AppPageHeader>

    <el-skeleton
      v-if="loading && !loaded"
      :rows="6"
      animated
      aria-busy="true"
    />
    <el-result
      v-else-if="error && !loaded"
      icon="error"
      :title="$t('common.loadFailed')"
      :sub-title="error"
    >
      <template #extra>
        <el-button
          type="primary"
          @click="$emit('refresh')"
        >
          {{ $t('common.retry') }}
        </el-button>
      </template>
    </el-result>

    <template v-else>
      <el-alert
        v-if="error"
        type="error"
        :title="error"
        show-icon
        :closable="false"
        class="load-alert"
      />
      <div
        v-loading="loading"
        :aria-busy="loading"
      >
        <slot />
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import AppPageHeader from '@/components/app/AppPageHeader.vue'

defineProps<{
  title: string
  subtitle: string
  loading: boolean
  loaded: boolean
  error: string
  showRefresh: boolean
}>()

defineEmits<{ refresh: [] }>()
</script>
