/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#22c55e',
        'primary-hover': '#16a34a',
        'bg-hero': '#0a0f1c',
        'bg-section': '#0f172a',
        'bg-footer': '#020617',
        'text-light': '#94a3b8',
        'text-dim': '#64748b',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
    },
  },
  plugins: [],
}
