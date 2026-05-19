<script lang="ts">
  import { router } from "svelte-spa-router";
  import { logout } from "../lib/stores/auth";
  import { theme, toggleTheme } from "../lib/stores/theme";

  export let sidebarOpen = false;

  const menuItems = [
    { path: "/overview", label: "Tổng Quan", icon: "📊" },
    { path: "/queue", label: "Tiến Trình Chạy", icon: "⚡" },
    { path: "/history", label: "Lịch Sử Tải", icon: "⏳" },
    { path: "/users", label: "Người Dùng", icon: "👥" },
    { path: "/logs", label: "Logs Hệ Thống", icon: "📝" },
    { path: "/broadcast", label: "Phát Thông Báo", icon: "📢" },
    { path: "/config", label: "Cấu Hình", icon: "⚙️" },
  ];

  function closeSidebar() {
    sidebarOpen = false;
  }
</script>

<!-- Mobile Sidebar Backdrop -->
{#if sidebarOpen}
  <!-- eslint-disable-next-line svelte/valid-compile -->
  <button
    aria-label="Close sidebar"
    class="fixed inset-0 z-40 bg-slate-950/40 backdrop-blur-sm lg:hidden transition-opacity duration-300"
    on:click={closeSidebar}
  ></button>
{/if}

<!-- Sidebar Navigation Drawer -->
<aside
  class="fixed top-0 bottom-0 left-0 z-40 w-64 glass lg:translate-x-0 transition-transform duration-300 ease-out flex flex-col border-r border-slate-200/10 bg-slate-950/70"
  class:-translate-x-full={!sidebarOpen}
>
  <!-- Brand logo header -->
  <div class="h-16 flex items-center px-6 gap-3 border-b border-slate-200/10">
    <div
      class="w-8 h-8 rounded-lg bg-gradient-to-tr from-sky-500 to-indigo-500 flex items-center justify-center text-white font-bold text-lg font-heading shadow-md shadow-sky-500/20"
    >
      ↓
    </div>
    <div>
      <h2 class="font-heading font-bold text-slate-100 leading-none">
        Download Bot
      </h2>
      <span
        class="text-[10px] text-slate-400 font-medium tracking-wider uppercase"
        >Quản Trị VPS</span
      >
    </div>
  </div>

  <!-- Navigation items -->
  <nav class="flex-grow p-4 space-y-1 overflow-y-auto">
    {#each menuItems as item}
      <a
        href={`#${item.path}`}
        class="flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all duration-200
        {router.location === item.path ||
        (router.location === '/' && item.path === '/overview')
          ? 'bg-sky-500/10 text-sky-400'
          : 'text-slate-400 hover:text-slate-200 hover:bg-slate-200/5'}"
        on:click={closeSidebar}
      >
        <span class="text-base">{item.icon}</span>
        <span>{item.label}</span>
      </a>
    {/each}
  </nav>

  <!-- Sidebar Footer -->
  <div class="p-4 border-t border-slate-200/10 space-y-2">
    <!-- Theme Toggler -->
    <button
      on:click={toggleTheme}
      class="w-full flex items-center justify-between px-4 py-2.5 rounded-xl text-sm font-medium text-slate-400 hover:text-slate-200 hover:bg-slate-200/5 transition-all duration-200"
    >
      <div class="flex items-center gap-3">
        <span>{$theme === "dark" ? "🌙" : "☀️"}</span>
        <span>Chế độ: {$theme === "dark" ? "Tối" : "Sáng"}</span>
      </div>
    </button>

    <!-- Logout -->
    <button
      on:click={logout}
      class="w-full flex items-center gap-3 px-4 py-2.5 rounded-xl text-sm font-medium text-rose-400 hover:text-rose-300 hover:bg-rose-500/10 transition-all duration-200"
    >
      <span>🚪</span>
      <span>Đăng xuất</span>
    </button>
  </div>
</aside>
