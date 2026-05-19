<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { api } from '../lib/api';
  import { showToast } from '../lib/stores/toast';
  import type { QueueItem } from '../lib/types';

  let queue: QueueItem[] = [];
  let loading = true;
  let pollInterval: any = null;

  async function fetchQueue() {
    try {
      queue = await api.getQueue();
    } catch (err: any) {
      showToast('error', err.message || 'Lỗi khi tải thông tin tiến trình.');
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchQueue();
    // Poll the API every 1.5 seconds for real-time progress bar feedback
    pollInterval = setInterval(fetchQueue, 1500);
  });

  onDestroy(() => {
    clearInterval(pollInterval);
  });

  function formatTime(startedAt: string): string {
    try {
      const start = new Date(startedAt);
      const now = new Date();
      const diffSec = Math.floor((now.getTime() - start.getTime()) / 1000);
      
      if (diffSec < 60) {
        return `${diffSec} giây trước`;
      }
      const diffMin = Math.floor(diffSec / 60);
      return `${diffMin} phút ${diffSec % 60} giây trước`;
    } catch {
      return startedAt;
    }
  }
</script>

{#if loading}
  <div class="h-full flex items-center justify-center p-8">
    <div class="flex flex-col items-center gap-3">
      <span class="w-8 h-8 rounded-full border-4 border-sky-500/20 border-t-sky-500 animate-spin"></span>
      <p class="text-sm font-semibold text-slate-500 dark:text-slate-400">Đang tải thông tin tiến trình chạy...</p>
    </div>
  </div>
{:else}
  <div class="p-6">
    <div class="glass p-6 rounded-2xl border border-slate-200/10">
      <div class="flex items-center justify-between mb-6">
        <div>
          <h3 class="font-heading font-bold text-lg text-slate-800 dark:text-slate-100">Tiến trình tải xuống hiện tại</h3>
          <p class="text-xs text-slate-400 mt-1">Tự động đồng bộ hóa trạng thái tiến trình thời gian thực mỗi 1.5s.</p>
        </div>
        <span class="px-3 py-1.5 rounded-full text-xs font-bold bg-sky-500/10 text-sky-500 border border-sky-500/20">
          ● Đang tải: {queue.length} luồng
        </span>
      </div>

      {#if queue.length === 0}
        <div class="py-16 flex flex-col items-center justify-center text-center">
          <span class="text-4xl mb-4">💤</span>
          <h4 class="font-bold text-slate-700 dark:text-slate-300">Không có tiến trình nào đang chạy</h4>
          <p class="text-xs text-slate-500 mt-1 max-w-xs">Hệ thống đang rảnh. Khi người dùng gửi link qua Telegram Bot, tiến trình tải sẽ xuất hiện tại đây.</p>
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
          {#each queue as item (item.id)}
            <div class="glass p-5 rounded-2xl border border-slate-200/10 hover:shadow-lg transition-all flex flex-col justify-between">
              <div>
                <div class="flex items-center justify-between mb-3">
                  <span class="px-2 py-0.5 rounded text-[10px] font-bold tracking-wider bg-sky-500/10 text-sky-500 border border-sky-500/20">
                    ID: {item.id}
                  </span>
                  <span class="text-[10px] font-semibold text-slate-400">
                    Bắt đầu: {formatTime(item.started_at)}
                  </span>
                </div>

                <h4 class="font-bold text-sm text-slate-800 dark:text-slate-100 truncate mb-1" title={item.title}>
                  {item.title}
                </h4>
                
                <a
                  href={item.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-xs text-sky-500 hover:underline truncate block mb-4"
                  title={item.url}
                >
                  {item.url}
                </a>
              </div>

              <!-- Progress bar container -->
              <div class="space-y-1.5">
                <div class="flex items-center justify-between text-xs font-bold text-slate-600 dark:text-slate-400">
                  <span>Trạng thái: Đang tải...</span>
                  <span class="text-sky-500 font-mono">{item.progress.toFixed(1)}%</span>
                </div>
                <!-- Track -->
                <div class="w-full h-2 rounded-full bg-slate-200 dark:bg-slate-800 overflow-hidden">
                  <!-- Filled Bar -->
                  <div
                    class="h-full rounded-full bg-gradient-to-r from-sky-500 to-indigo-500 transition-all duration-300 ease-out"
                    style={`width: ${item.progress}%`}
                  ></div>
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{/if}
