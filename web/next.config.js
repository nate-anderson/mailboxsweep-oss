const withPWA = require('next-pwa')({
  dest: 'public',
});

/** @type {import('next').NextConfig} */
const nextConfig = withPWA({
  reactStrictMode: true,
  output: 'export',
  env: {
    MAILBOXSWEEP_API_HOST: process.env.MAILBOXSWEEP_API_HOST, 
    MAILBOXSWEEP_API_PORT: process.env.MAILBOXSWEEP_API_PORT,
    ENV: process.env.ENV
  },
  images:{
    unoptimized: true
  },
  trailingSlash: true,
});

module.exports = nextConfig;
