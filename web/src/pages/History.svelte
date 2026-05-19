<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api";
  import { showToast } from "../lib/stores/toast";
  import type { DownloadHistory } from "../lib/types";

  let history: DownloadHistory[] = [];
  let loading = true;

  // Retrieve API server base URL for direct downloads
  const isDev = import.meta.env.DEV;
  const SERVER_HOST = isDev ? "http://localhost:8080" : "";

  async function loadHistory() {
    try {
      history = await api.getHistory();
    } catch (err: any) {
      showToast("error", err.message || "Lỗi khi tải lịch sử.");
    } finally {
      loading = false;
    }
  }

  onMount(loadHistory);

  async function handleDelete(id: number) {
    if (
      !confirm(
        "Bạn có chắc chắn muốn xóa bản ghi và tệp vật lý này khỏi máy chủ?",
      )
    )
      return;

    try {
      await api.deleteRecord(id);
      showToast("success", "Đã xóa bản ghi thành công!");
      // Reload history list
      await loadHistory();
    } catch (err: any) {
      showToast("error", err.message || "Lỗi khi xóa bản ghi.");
    }
  }

  function formatBytes(bytes: number): string {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  }

  function formatDate(dateStr: string): string {
    try {
      const d = new Date(dateStr);
      return d.toLocaleString("vi-VN");
    } catch (err) {
      return dateStr;
    }
  }
</script>

{#if loading}
  <div class="h-full flex items-center justify-center p-8">
    <div class="flex flex-col items-center gap-3">
      <span
        class="w-8 h-8 rounded-full border-4 border-sky-500/20 border-t-sky-500 animate-spin"
      ></span>
      <p class="text-sm font-semibold text-slate-500 dark:text-slate-400">
        Đang tải lịch sử hệ thống...
      </p>
    </div>
  </div>
{:else}
  <div class="p-6">
    <div class="glass p-6 rounded-2xl border border-slate-200/10">
      <div class="flex items-center justify-between mb-6">
        <h3
          class="font-heading font-bold text-lg text-slate-800 dark:text-slate-100"
        >
          Bản ghi lượt tải hệ thống
        </h3>
        <button
          on:click={() => {
            loading = true;
            loadHistory();
          }}
          class="px-4 py-2 rounded-xl text-xs font-bold bg-slate-200 dark:bg-slate-800 text-slate-700 dark:text-slate-200 hover:opacity-90 active:scale-[0.98] transition-all"
        >
          🔄 Làm mới
        </button>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left border-collapse">
          <thead>
            <tr
              class="border-b border-slate-200/10 text-slate-400 uppercase tracking-wider text-[10px] font-semibold"
            >
              <th class="pb-3">Nền tảng</th>
              <th class="pb-3">Tiêu đề / Link gốc</th>
              <th class="pb-3">User ID</th>
              <th class="pb-3">Định dạng</th>
              <th class="pb-3">Dung lượng</th>
              <th class="pb-3">Thời gian</th>
              <th class="pb-3">Tệp vật lý</th>
              <th class="pb-3 text-right">Thao tác</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-200/5 text-sm">
            {#each history as h (h.id)}
              <tr class="text-slate-700 dark:text-slate-300">
                <!-- Platform badge -->
                <td class="py-3.5">
                  <span
                    class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-slate-200 dark:bg-slate-800 text-slate-700 dark:text-slate-300"
                  >
                    {h.platform}
                  </span>
                </td>

                <!-- Title & Original link -->
                <td class="py-3.5 max-w-[280px]">
                  <div
                    class="font-semibold text-slate-800 dark:text-slate-100 truncate mb-0.5"
                    title={h.title}
                  >
                    {h.title}
                  </div>
                  <a
                    href={h.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="text-xs text-sky-500 hover:underline truncate block"
                    title={h.url}>{h.url}</a
                  >
                </td>

                <!-- User ID -->
                <td class="py-3.5 font-mono text-slate-500">{h.user_id}</td>

                <!-- Format -->
                <td
                  class="py-3.5 font-semibold text-slate-600 dark:text-slate-400"
                  >{h.format}</td
                >

                <!-- Size -->
                <td class="py-3.5 font-bold text-slate-700 dark:text-slate-200"
                  >{formatBytes(h.file_size)}</td
                >

                <!-- Date -->
                <td class="py-3.5 text-xs text-slate-500"
                  >{formatDate(h.created_at)}</td
                >

                <!-- Physical exist status -->
                <td class="py-3.5">
                  {#if h.file_exist}
                    <span
                      class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-500 border border-emerald-500/20"
                    >
                      ● Còn tồn tại
                    </span>
                  {:else}
                    <span
                      class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-rose-500/10 text-rose-500 border border-rose-500/20"
                    >
                      ● Đã dọn dẹp
                    </span>
                  {/if}
                </td>

                <!-- Action buttons -->
                <td class="py-3.5 text-right space-x-2">
                  {#if h.file_exist && h.download_url}
                    <a
                      href={`${SERVER_HOST}${h.download_url}`}
                      target="_blank"
                      download
                      class="px-2.5 py-1.5 rounded-lg text-xs font-bold bg-sky-500/10 text-sky-500 border border-sky-500/20 hover:bg-sky-500 hover:text-white transition-all inline-block align-middle"
                    >
                      💾 Tải về
                    </a>
                  {/if}
                  <button
                    on:click={() => handleDelete(h.id)}
                    class="px-2.5 py-1.5 rounded-lg text-xs font-bold bg-rose-500/10 text-rose-500 border border-rose-500/20 hover:bg-rose-500 hover:text-white transition-all align-middle"
                  >
                    🗑 Xóa
                  </button>
                </td>
              </tr>
            {/each}
            {#if history.length === 0}
              <tr>
                <td colspan="8" class="py-12 text-center text-slate-400"
                  >Không có lượt tải nào ghi nhận trong hệ thống.</td
                >
              </tr>
            {/if}
          </tbody>
        </table>
      </div>
    </div>
  </div>
{/if}
