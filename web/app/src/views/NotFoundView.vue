<template>
  <main class="not-found">
    <el-card
      class="panel"
      shadow="never"
    >
      <p class="code">
        404
      </p>
      <h1>{{ $t('notFound.title') }}</h1>
      <p class="description">
        {{ $t('notFound.description') }}
      </p>
      <div class="actions">
        <el-button
          type="primary"
          @click="router.push('/')"
        >
          {{ $t('notFound.backHome') }}
        </el-button>
        <el-button @click="router.back()">
          {{ $t('notFound.back') }}
        </el-button>
      </div>
    </el-card>
  </main>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<style scoped>
.not-found {
  min-height: 100dvh;
  display: grid;
  place-items: center;
  padding: 24px;
  background: var(--liki-paper);
}
.panel {
  width: min(520px, 100%);
  border-radius: var(--radius-card);
  text-align: center;
  box-shadow: var(--shadow-card);
}
.code {
  margin: 0;
  color: var(--liki-primary);
  font-size: 40px;
  font-weight: 700;
  line-height: 1;
  letter-spacing: -0.04em;
}
h1 {
  margin: 12px 0 8px;
  font-size: var(--font-size-title-lg);
  font-weight: 650;
  letter-spacing: -0.03em;
}
.description {
  margin: 0 0 24px;
  color: var(--liki-text-2);
}
.actions {
  display: flex;
  justify-content: center;
  gap: 12px;
  flex-wrap: wrap;
}
</style>
