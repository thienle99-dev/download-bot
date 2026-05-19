<script lang="ts">
  import Router, { router, push } from "svelte-spa-router";
  import { isAuthenticated } from "./lib/stores/auth";
  import { theme } from "./lib/stores/theme";
  import Sidebar from "./components/Sidebar.svelte";
  import Header from "./components/Header.svelte";
  import Toast from "./components/Toast.svelte";

  // Import Pages
  import Login from "./pages/Login.svelte";
  import Overview from "./pages/Overview.svelte";
  import Queue from "./pages/Queue.svelte";
  import History from "./pages/History.svelte";
  import Users from "./pages/Users.svelte";
  import Logs from "./pages/Logs.svelte";
  import Broadcast from "./pages/Broadcast.svelte";
  import Config from "./pages/Config.svelte";

  // Define SPA routes
  const routes = {
    "/": Overview,
    "/login": Login,
    "/overview": Overview,
    "/queue": Queue,
    "/history": History,
    "/users": Users,
    "/logs": Logs,
    "/broadcast": Broadcast,
    "/config": Config,
  };

  let sidebarOpen = false;

  // SPA Route Guards
  $: {
    if (!$isAuthenticated && router.location !== "/login") {
      push("/login");
    } else if ($isAuthenticated && router.location === "/login") {
      push("/overview");
    }
  }
</script>

<!-- Global Toast notifications floating container -->
<Toast />

<div
  class="min-h-screen text-slate-700 bg-slate-50 dark:text-slate-300 dark:bg-slate-950 transition-colors duration-200"
>
  {#if router.location === "/login"}
    <!-- Full-screen login layout with direct Router render -->
    <main class="min-h-screen">
      <Router {routes} />
    </main>
  {:else}
    <!-- Main Dashboard layout wrapper -->
    <div class="min-h-screen flex">
      <!-- Responsive Sidebar Drawer -->
      <Sidebar bind:sidebarOpen />

      <!-- Main Content viewport panel -->
      <div class="flex-grow lg:pl-64 flex flex-col min-h-screen">
        <!-- Top bar Header -->
        <Header bind:sidebarOpen />

        <!-- Component content page viewports -->
        <main class="flex-grow">
          <Router {routes} />
        </main>
      </div>
    </div>
  {/if}
</div>
