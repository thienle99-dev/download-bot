<script lang="ts">
  import { router } from 'svelte-spa-router';

  export let sidebarOpen = false;

  const pageTitles: Record<string, string> = {
    '/overview': 'Tổng Quan Hệ Thống',
    '/queue': 'Tiến Trình Đang Chạy',
    '/history': 'Lịch Sử Tải Xuống',
    '/users': 'Quản Lý Người Dùng',
    '/logs': 'Logs Hệ Thống (Real-time)',
    '/broadcast': 'Gửi Tin Nhắn Đồng Loạt',
    '/config': 'Cấu Hình Hệ Thống',
  };

  $: title = pageTitles[router.location] || 'Tổng Quan Hệ Thống';

  function toggleSidebar() {
    sidebarOpen = !sidebarOpen;
  }
</script>

<header class="h-16 flex items-center justify-between px-6 border-b border-slate-200/10 bg-slate-900/50 dark:bg-slate-950/40 backdrop-blur-md sticky top-0 z-30 transition-all duration-200">
  <div class="flex items-center gap-4">
    <!-- Hamburger button for mobile -->
    <button
      on:click={toggleSidebar}
      class="p-2 -ml-2 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-200/5 lg:hidden focus:outline-none"
      aria-label="Toggle menu"
    >
      <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
      </svg>
    </button>

    <h1 class="font-heading font-bold text-xl text-slate-800 dark:text-slate-100 transition-colors">
      {title}
    </h1>
  </div>

  <!-- Server Status Indicator -->
  <div class="flex items-center gap-3">
    <div class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/25">
      <span class="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
      <span class="text-xs font-semibold text-emerald-500 uppercase tracking-wider">VPS Connected</span>
    </div>
  </div>
</header>
