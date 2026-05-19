<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../lib/api';
  import { showToast } from '../lib/stores/toast';
  import type { SystemConfig } from '../lib/types';

  let config: SystemConfig | null = null;
  let loading = true;

  async function loadConfig() {
    try {
      config = await api.getConfig();
    } catch (err: any) {
      showToast('error', err.message || 'Lỗi khi tải thông tin cấu hình.');
    } finally {
      loading = false;
    }
  }

  onMount(loadConfig);
</script>

{#if loading}
  <div class="h-full flex items-center justify-center p-8">
    <div class="flex flex-col items-center gap-3">
      <span class="w-8 h-8 rounded-full border-4 border-sky-500/20 border-t-sky-500 animate-spin"></span>
      <p class="text-sm font-semibold text-slate-500 dark:text-slate-400">Đang tải thông số cấu hình VPS...</p>
    </div>
  </div>
{:else}
  <div class="p-6">
    <div class="glass p-6 rounded-2xl border border-slate-200/10 max-w-2xl">
      <h3 class="font-heading font-bold text-lg text-slate-800 dark:text-slate-100 mb-2">Cấu hình VPS</h3>
      <p class="text-xs text-slate-500 mb-6">
        Thông số kỹ thuật hoạt động của hệ thống được trích xuất trực tiếp từ tệp `.env` trên VPS.
      </p>

      <div class="space-y-4">
        <!-- DB Path -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between p-4 rounded-xl bg-slate-900/40 border border-slate-800/50">
          <span class="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1 sm:mb-0">Đường dẫn SQLite DB</span>
          <span class="font-mono text-sm text-slate-200">{config?.db_path}</span>
        </div>

        <!-- Download folder -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between p-4 rounded-xl bg-slate-900/40 border border-slate-800/50">
          <span class="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1 sm:mb-0">Thư mục Downloads</span>
          <span class="font-mono text-sm text-slate-200">{config?.download_dir}</span>
        </div>

        <!-- Cache folder -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between p-4 rounded-xl bg-slate-900/40 border border-slate-800/50">
          <span class="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1 sm:mb-0">Thư mục Cache</span>
          <span class="font-mono text-sm text-slate-200">{config?.cache_dir}</span>
        </div>

        <!-- Max concurrency -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between p-4 rounded-xl bg-slate-900/40 border border-slate-800/50">
          <span class="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1 sm:mb-0">Số luồng tải tối đa</span>
          <span class="font-bold text-sm text-sky-400">{config?.max_concurrent} tiến trình đồng thời</span>
        </div>

        <!-- Public URL -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between p-4 rounded-xl bg-slate-900/40 border border-slate-800/50">
          <span class="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1 sm:mb-0">Công khai URL (Public Link)</span>
          <span class="font-mono text-sm text-sky-500">{config?.public_url}</span>
        </div>

        <!-- Server port -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between p-4 rounded-xl bg-slate-900/40 border border-slate-800/50">
          <span class="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1 sm:mb-0">Cổng mạng Web Server</span>
          <span class="font-mono text-sm text-slate-200">{config?.server_port}</span>
        </div>
      </div>
    </div>
  </div>
{/if}
