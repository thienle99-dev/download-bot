import { writable } from 'svelte/store';

// Retrieve initial theme preference
const initialTheme = localStorage.getItem('theme') || 'dark';

export const theme = writable<string>(initialTheme);

// Subscribe to theme store and update document body/classes
theme.subscribe((value) => {
  localStorage.setItem('theme', value);
  if (value === 'light') {
    document.documentElement.classList.add('light');
    document.documentElement.classList.remove('dark');
  } else {
    document.documentElement.classList.add('dark');
    document.documentElement.classList.remove('light');
  }
});

export function toggleTheme() {
  theme.update((current) => (current === 'light' ? 'dark' : 'light'));
}
