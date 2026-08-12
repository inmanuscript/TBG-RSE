/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts}'],
  theme: {
    extend: {
      colors: {
        surface: {
          DEFAULT: '#141a22',
          raised: '#1c2430',
          border: '#2a3544',
        },
        ink: {
          DEFAULT: '#e8eef6',
          muted: '#8b9bb0',
        },
        mars: {
          DEFAULT: '#d4652f',
          dim: '#a34d22',
          glow: '#f08a4b',
        },
      },
      fontFamily: {
        display: ['"Orbitron"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        body: ['"IBM Plex Sans"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
      },
      boxShadow: {
        toast: '0 12px 40px rgba(0,0,0,0.45)',
      },
      keyframes: {
        slideIn: {
          '0%': { opacity: '0', transform: 'translateX(24px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        pulseReady: {
          '0%, 100%': { boxShadow: '0 0 0 0 rgba(212,101,47,0.45)' },
          '50%': { boxShadow: '0 0 0 8px rgba(212,101,47,0)' },
        },
        fadeUp: {
          '0%': { opacity: '0', transform: 'translateY(8px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
      },
      animation: {
        slideIn: 'slideIn 0.28s ease-out',
        pulseReady: 'pulseReady 1.6s ease-in-out infinite',
        fadeUp: 'fadeUp 0.35s ease-out',
      },
    },
  },
  plugins: [],
}
