<script lang="ts">
  import { onMount, onDestroy, afterUpdate } from 'svelte';
  import { logsStore, logConnectionState, connectLogs, disconnectLogs, clearLogsStore } from '../lib/stores/logs';

  let terminalElement: HTMLDivElement;
  let autoScroll = true;
  let searchQuery = '';
  let selectedLevel = 'ALL';

  onMount(() => {
    connectLogs();
  });

  onDestroy(() => {
    disconnectLogs();
  });

  // Keep terminal scrolled to bottom if autoScroll is active
  afterUpdate(() => {
    if (autoScroll && terminalElement) {
      terminalElement.scrollTop = terminalElement.scrollHeight;
    }
  });

  // Filter logs reactively
  $: filteredLogs = $logsStore.filter((log) => {
    const matchesSearch = log.message.toLowerCase().includes(searchQuery.toLowerCase()) || 
                          log.level.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesLevel = selectedLevel === 'ALL' || log.level === selectedLevel;
    return matchesSearch && matchesLevel;
  });

  function formatDate(timestamp: string): string {
    try {
      const d = new Date(timestamp);
      return d.toLocaleTimeString('vi-VN') + '.' + String(d.getMilliseconds()).padStart(3, '0');
    } catch {
      return timestamp;
    }
  }
</script>

<div class="p-6 flex flex-col h-[calc(100vh-5rem)]">
  <!-- Terminal controls -->
  <div class="glass p-4 rounded-2xl border border-slate-200/10 mb-4 flex flex-wrap items-center justify-between gap-4">
    <div class="flex flex-wrap items-center gap-3">
      <!-- Search input -->
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="🔍 Tìm kiếm log..."
        class="px-4 py-2 rounded-xl text-xs bg-slate-900 border border-slate-800 text-slate-100 focus:outline-none focus:border-sky-500 w-48 font-medium"
      />

      <!-- Level filter -->
      <select
        bind:value={selectedLevel}
        class="px-3 py-2 rounded-xl text-xs bg-slate-900 border border-slate-800 text-slate-300 focus:outline-none focus:border-sky-500 font-semibold"
      >
        <option value="ALL">📋 Tất cả cấp độ</option>
        <option value="INFO">🟢 Cấp INFO</option>
        <option value="WARN">🟡 Cấp WARNING</option>
        <option value="ERROR">🔴 Cấp ERROR</option>
      </select>
    </div>

    <div class="flex items-center gap-3">
      <!-- Auto Scroll Switch -->
      <label class="flex items-center gap-2 cursor-pointer select-none">
        <input type="checkbox" bind:checked={autoScroll} class="sr-only peer" />
        <div class="w-8 h-4 bg-slate-800 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-3 after:w-3 after:transition-all peer-checked:bg-sky-500 relative"></div>
        <span class="text-xs font-semibold text-slate-400">Tự động cuộn</span>
      </label>

      <!-- Clear Screen -->
      <button
        on:click={clearLogsStore}
        class="px-4 py-2 rounded-xl text-xs font-bold bg-slate-200 dark:bg-slate-800 text-slate-700 dark:text-slate-200 hover:opacity-90 active:scale-[0.98] transition-all"
      >
        🗑 Xóa màn hình
      </button>

      <!-- Reconnection Indicator -->
      {#if $logConnectionState === 'connected'}
        <span class="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-bold uppercase tracking-wider bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
          ● Đang kết nối
        </span>
      {:else if $logConnectionState === 'connecting'}
        <span class="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-bold uppercase tracking-wider bg-amber-500/10 text-amber-500 border border-amber-500/20 animate-pulse">
          ⚡ Đang thử lại...
        </span>
      {:else}
        <button
          on:click={connectLogs}
          class="px-2.5 py-1 rounded-full text-[10px] font-bold uppercase tracking-wider bg-rose-500/10 text-rose-500 border border-rose-500/20 hover:bg-rose-500 hover:text-white transition-all"
        >
          ✕ Mất kết nối (Click thử lại)
        </button>
      {/if}
    </div>
  </div>

  <!-- Terminal body -->
  <div
    bind:this={terminalElement}
    class="flex-grow bg-black border border-slate-900 rounded-2xl p-6 overflow-y-auto font-mono text-xs leading-relaxed shadow-2xl relative"
  >
    <div class="space-y-1.5">
      {#each filteredLogs as log}
        <div class="flex items-start gap-3 hover:bg-slate-900/50 py-0.5 rounded transition-all">
          <!-- Time -->
          <span class="text-slate-600 select-none flex-shrink-0">{formatDate(log.timestamp)}</span>
          
          <!-- Level Badge -->
          <span
            class="px-1.5 py-0.5 rounded-[3px] text-[8px] font-extrabold uppercase tracking-wider flex-shrink-0 
            {log.level === 'INFO' ? 'bg-sky-500/10 text-sky-400' : ''} 
            {log.level === 'WARN' ? 'bg-amber-500/10 text-amber-400' : ''} 
            {log.level === 'ERROR' ? 'bg-rose-500/10 text-rose-400' : ''}"
          >
            {log.level}
          </span>

          <!-- Log Content message -->
          <span class="text-slate-300 break-all select-text">{log.message}</span>
        </div>
      {/each}

      {#if filteredLogs.length === 0}
        <div class="h-64 flex items-center justify-center text-slate-600 italic select-none">
          Chưa nhận được log sự kiện nào...
        </div>
      {/if}
    </div>
  </div>
</div>
