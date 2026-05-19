<script lang="ts">
  export let title: string = "";
  export let value: string | number = "";
  export let icon: string = "";
  export let description: string = "";
  export let color: "sky" | "emerald" | "violet" | "amber" = "sky";
</script>

<div
  class="glass p-6 rounded-2xl flex flex-col justify-between relative overflow-hidden transition-all duration-300 hover:translate-y-[-2px] hover:shadow-xl group"
>
  <!-- Glowing colored accent glow -->
  <div
    class="absolute -right-4 -top-4 w-24 h-24 rounded-full blur-2xl opacity-10 group-hover:opacity-20 transition-opacity duration-300"
    class:bg-sky-500={color === "sky"}
    class:bg-emerald-500={color === "emerald"}
    class:bg-violet-500={color === "violet"}
    class:bg-amber-500={color === "amber"}
  ></div>

  <div class="flex items-start justify-between">
    <div>
      <p
        class="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-1"
      >
        {title}
      </p>
      <h3
        class="font-heading font-extrabold text-3xl text-slate-800 dark:text-slate-100 tracking-tight leading-none"
      >
        {value}
      </h3>
    </div>
    <div
      class="w-12 h-12 rounded-xl flex items-center justify-center text-xl shadow-inner border border-slate-200/10
      {color === 'sky' ? 'bg-sky-500/10' : ''} 
      {color === 'emerald' ? 'bg-emerald-500/10' : ''} 
      {color === 'violet' ? 'bg-violet-500/10' : ''} 
      {color === 'amber' ? 'bg-amber-500/10' : ''}"
    >
      {icon}
    </div>
  </div>

  {#if description}
    <p
      class="text-xs text-slate-400 dark:text-slate-500 mt-4 font-medium flex items-center gap-1"
    >
      {description}
    </p>
  {/if}
</div>
