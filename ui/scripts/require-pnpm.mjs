// Keep installation reproducible without downloading a package-manager guard.
if (!process.env.npm_config_user_agent?.startsWith('pnpm/')) {
  console.error('This project uses pnpm. Run: pnpm install --frozen-lockfile');
  process.exit(1);
}
