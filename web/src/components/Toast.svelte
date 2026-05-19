<script lang="ts">
  import { toasts } from '../lib/stores/toast';
</script>

<div class="fixed top-4 right-4 z-50 flex flex-col gap-2 max-w-sm w-full pointer-events-none">
  {#each $toasts as toast (toast.id)}
    <div
      class="glass p-4 rounded-xl shadow-2xl flex items-center gap-3 animate-slide-in pointer-events-auto border transition-all duration-300"
      class:border-emerald-500={toast.type === 'success'}
      class:border-rose-500={toast.type === 'error'}
      class:border-sky-500={toast.type === 'info'}
    >
      <div class="flex-shrink-0">
        {#if toast.type === 'success'}
          <span class="text-emerald-500 text-xl">✓</span>
        {:else if toast.type === 'error'}
          <span class="text-rose-500 text-xl">✕</span>
        {:else}
          <span class="text-sky-500 text-xl">ℹ</span>
        {/if}
      </div>
      <div class="text-sm font-medium text-slate-800 dark:text-slate-200">
        {toast.message}
      </div>
    </div>
  {/each}
</div>

<style>
  @keyframes slideIn {
    from {
      transform: translateX(100%) translateY(-10px);
      opacity: 0;
    }
    to {
      transform: translateX(0) translateY(0);
      opacity: 1;
    }
  }

  .animate-slide-in {
    animation: slideIn 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }
</style>
