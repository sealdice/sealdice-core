import tailwindcss from 'tailwindcss';
import autoprefixer from 'autoprefixer';
import postcssColorMix from '@csstools/postcss-color-mix-function';

export default {
  plugins: [
    tailwindcss(),
    postcssColorMix({
      preserve: true,
    }),
    autoprefixer(),
  ],
};
