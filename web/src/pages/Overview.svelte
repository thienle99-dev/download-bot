<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api";
  import { showToast } from "../lib/stores/toast";
  import StatCard from "../components/StatCard.svelte";
  import type { Stats, DownloadHistory } from "../lib/types";

  let stats: Stats | null = null;
  let recentDownloads: DownloadHistory[] = [];
  let loading = true;

  onMount(async () => {
    try {
      const [statsData, historyData] = await Promise.all([
        api.getStats(),
        api.getHistory(),
      ]);
      stats = statsData;
      // Show only last 5 downloads for dashboard glance
      recentDownloads = historyData.slice(0, 5);
    } catch (err: any) {
      showToast("error", err.message || "Lỗi khi tải thông tin hệ thống.");
    } finally {
      loading = false;
    }
  });

  function formatBytes(bytes: number): string {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const dm = 2;
    const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
  }

  // Analytics helper calculations
  $: ytCount = recentDownloads.filter((h) => h.platform === "youtube").length;
  $: ttCount = recentDownloads.filter((h) => h.platform === "tiktok").length;
  $: ytPercent = recentDownloads.length
    ? Math.round((ytCount / recentDownloads.length) * 100)
    : 50;
  $: ttPercent = recentDownloads.length
    ? Math.round((ttCount / recentDownloads.length) * 100)
    : 50;
</script>

{#if loading}
  <div class="h-full flex items-center justify-center p-8">
    <div class="flex flex-col items-center gap-3">
      <span
        class="w-8 h-8 rounded-full border-4 border-sky-500/20 border-t-sky-500 animate-spin"
      ></span>
      <p class="text-sm font-semibold text-slate-500 dark:text-slate-400">
        Đang tải dữ liệu Tổng Quan...
      </p>
    </div>
  </div>
{:else}
  <div class="p-6 space-y-6">
    <!-- Grid KPI Cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      <StatCard
        title="Tổng lượt tải"
        value={stats?.total_downloads || 0}
        icon="📥"
        description="Lượt tải lưu trong hệ thống"
        color="sky"
      />
      <StatCard
        title="Người dùng độc nhất"
        value={stats?.total_users || 0}
        icon="👥"
        description="Người dùng Telegram Bot"
        color="emerald"
      />
      <StatCard
        title="Bộ nhớ Cache đã dùng"
        value={formatBytes(stats?.storage_used || 0)}
        icon="📁"
        description="Dung lượng tệp lưu trên VPS"
        color="violet"
      />
      <StatCard
        title="Tiến trình đồng thời"
        value={`${stats?.max_concurrent || 3} luồng`}
        icon="⚙️"
        description="Giới hạn tối đa VPS tải về"
        color="amber"
      />
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Chart Distribution -->
      <div
        class="glass p-6 rounded-2xl flex flex-col justify-between lg:col-span-1 border border-slate-200/10"
      >
        <h3
          class="font-heading font-bold text-lg text-slate-800 dark:text-slate-100 mb-6"
        >
          Tỷ trọng Nền Tảng
        </h3>

        <div class="flex flex-col items-center gap-6">
          <!-- Beautiful minimalist SVG Donut chart -->
          <svg class="w-32 h-32 transform -rotate-90" viewBox="0 0 36 36">
            <!-- Background circle -->
            <circle
              class="text-slate-200 dark:text-slate-800"
              stroke="currentColor"
              stroke-width="4"
              fill="transparent"
              r="16"
              cx="18"
              cy="18"
            />
            <!-- YouTube Segment (Sky blue) -->
            <circle
              class="text-sky-500"
              stroke="currentColor"
              stroke-dasharray={`${ytPercent} 100`}
              stroke-width="4.2"
              stroke-linecap="round"
              fill="transparent"
              r="16"
              cx="18"
              cy="18"
            />
          </svg>

          <!-- Legends and counters -->
          <div class="w-full space-y-3">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <span class="w-3 h-3 rounded-full bg-sky-500"></span>
                <span
                  class="text-xs font-semibold text-slate-500 dark:text-slate-400"
                  >YouTube</span
                >
              </div>
              <span class="text-sm font-bold text-slate-700 dark:text-slate-200"
                >{ytPercent}%</span
              >
            </div>
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <span class="w-3 h-3 rounded-full bg-slate-400"></span>
                <span
                  class="text-xs font-semibold text-slate-500 dark:text-slate-400"
                  >Khác / TikTok</span
                >
              </div>
              <span class="text-sm font-bold text-slate-700 dark:text-slate-200"
                >{ttPercent}%</span
              >
            </div>
          </div>
        </div>
      </div>

      <!-- Recent Downloads Log Panel -->
      <div
        class="glass p-6 rounded-2xl lg:col-span-2 border border-slate-200/10 flex flex-col justify-between"
      >
        <div class="flex items-center justify-between mb-4">
          <h3
            class="font-heading font-bold text-lg text-slate-800 dark:text-slate-100"
          >
            Lượt Tải Gần Đây
          </h3>
          <a
            href="#/history"
            class="text-xs font-bold text-sky-500 hover:text-sky-400 transition-colors uppercase tracking-wider"
            >Xem tất cả →</a
          >
        </div>

        <div class="overflow-x-auto">
          <table class="w-full text-left border-collapse">
            <thead>
              <tr
                class="border-b border-slate-200/10 text-slate-400 uppercase tracking-wider text-[10px] font-semibold"
              >
                <th class="pb-3">Nền tảng</th>
                <th class="pb-3">Tiêu đề</th>
                <th class="pb-3">Định dạng</th>
                <th class="pb-3 text-right">Dung lượng</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-200/5 text-sm">
              {#each recentDownloads as h}
                <tr class="text-slate-700 dark:text-slate-300">
                  <td class="py-3">
                    <span
                      class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-slate-200 dark:bg-slate-800 text-slate-700 dark:text-slate-300"
                    >
                      {h.platform}
                    </span>
                  </td>
                  <td
                    class="py-3 max-w-[200px] truncate font-medium text-slate-800 dark:text-slate-100"
                    >{h.title}</td
                  >
                  <td class="py-3 font-semibold text-slate-500">{h.format}</td>
                  <td class="py-3 text-right font-bold"
                    >{formatBytes(h.file_size)}</td
                  >
                </tr>
              {/each}
              {#if recentDownloads.length === 0}
                <tr>
                  <td colspan="4" class="py-8 text-center text-slate-400"
                    >Không có lượt tải gần đây.</td
                  >
                </tr>
              {/if}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
{/if}
