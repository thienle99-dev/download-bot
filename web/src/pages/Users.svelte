<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "../lib/api";
  import { showToast } from "../lib/stores/toast";
  import type { UserStat } from "../lib/types";

  let users: UserStat[] = [];
  let loading = true;

  async function loadUsers() {
    try {
      users = await api.getUsers();
    } catch (err: any) {
      showToast("error", err.message || "Lỗi khi tải danh sách người dùng.");
    } finally {
      loading = false;
    }
  }

  onMount(loadUsers);

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
        Đang tải thống kê người dùng...
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
          Bảng thống kê người dùng
        </h3>
        <button
          on:click={() => {
            loading = true;
            loadUsers();
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
              <th class="pb-3">User ID</th>
              <th class="pb-3">Chat ID</th>
              <th class="pb-3">Tổng lượt tải thành công</th>
              <th class="pb-3">Lần hoạt động cuối cùng</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-200/5 text-sm">
            {#each users as u}
              <tr class="text-slate-700 dark:text-slate-300">
                <!-- User ID -->
                <td
                  class="py-4 font-mono font-semibold text-slate-800 dark:text-slate-100"
                  >{u.user_id}</td
                >

                <!-- Chat ID -->
                <td class="py-4 font-mono text-slate-500">{u.chat_id}</td>

                <!-- Count -->
                <td class="py-4 font-bold text-sky-500">
                  {u.download_count} lượt tải
                </td>

                <!-- Date -->
                <td class="py-4 text-xs text-slate-500"
                  >{formatDate(u.last_download)}</td
                >
              </tr>
            {/each}
            {#if users.length === 0}
              <tr>
                <td colspan="4" class="py-12 text-center text-slate-400"
                  >Không có dữ liệu người dùng hoạt động.</td
                >
              </tr>
            {/if}
          </tbody>
        </table>
      </div>
    </div>
  </div>
{/if}
