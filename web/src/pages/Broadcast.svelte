<script lang="ts">
  import { api } from '../lib/api';
  import { showToast } from '../lib/stores/toast';

  let message = '';
  let loading = false;
  let result: { success: number; failed: number; total: number } | null = null;

  async function handleBroadcast() {
    if (!message.trim()) {
      showToast('error', 'Vui lòng nhập nội dung tin nhắn!');
      return;
    }

    if (!confirm('Bạn có chắc muốn gửi tin nhắn này tới tất cả người dùng trong hệ thống Telegram?')) return;

    loading = true;
    result = null;
    try {
      const res = await api.broadcast(message);
      result = res;
      showToast('success', 'Tin nhắn phát sóng đã hoàn tất!');
      message = '';
    } catch (err: any) {
      showToast('error', err.message || 'Gửi tin nhắn phát sóng thất bại.');
    } finally {
      loading = false;
    }
  }
</script>

<div class="p-6">
  <div class="glass p-6 rounded-2xl border border-slate-200/10 max-w-2xl">
    <h3 class="font-heading font-bold text-lg text-slate-800 dark:text-slate-100 mb-2">Phát thông báo đồng loạt</h3>
    <p class="text-xs text-slate-500 mb-6">
      Tin nhắn sẽ được gửi song song và trực tiếp đến tất cả người dùng đã từng kích hoạt và tải file từ Telegram Bot của bạn.
    </p>

    {#if result}
      <div class="p-4 rounded-xl border mb-6 bg-sky-500/10 border-sky-500/20 text-slate-800 dark:text-slate-200">
        <h4 class="font-bold text-sm mb-2">📊 Kết quả phát sóng gần nhất:</h4>
        <ul class="text-xs space-y-1 font-semibold">
          <li>● Thành công: <span class="text-emerald-500">{result.success}</span></li>
          <li>● Thất bại: <span class="text-rose-500">{result.failed}</span></li>
          <li>● Tổng số người dùng đích: <span class="text-sky-500">{result.total}</span></li>
        </ul>
      </div>
    {/if}

    <form on:submit|preventDefault={handleBroadcast} class="space-y-4">
      <div>
        <label for="message" class="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Nội dung tin nhắn</label>
        <textarea
          id="message"
          bind:value={message}
          rows="6"
          placeholder="Nhập nội dung tin nhắn gửi tới tất cả người dùng..."
          class="w-full px-4 py-3.5 rounded-xl bg-slate-900 border border-slate-800 text-slate-100 placeholder-slate-600 focus:outline-none focus:border-sky-500 focus:ring-1 focus:ring-sky-500 transition-all font-medium text-sm"
        ></textarea>
      </div>

      <button
        type="submit"
        disabled={loading}
        class="px-6 py-3.5 rounded-xl bg-gradient-to-r from-sky-500 to-indigo-500 hover:from-sky-400 hover:to-indigo-400 text-white font-semibold shadow-lg shadow-sky-500/20 active:scale-[0.98] transition-all disabled:opacity-50 disabled:pointer-events-none text-sm"
      >
        {loading ? '⏳ Đang phát sóng tin nhắn...' : '📢 Phát sóng ngay'}
      </button>
    </form>
  </div>
</div>
