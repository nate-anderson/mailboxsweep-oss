/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'export',
  env: {
    MAILBOXSWEEP_API_HOST: process.env.MAILBOXSWEEP_API_HOST, 
    MAILBOXSWEEP_API_PORT: process.env.MAILBOXSWEEP_API_PORT
  },
  images:{
    unoptimized: true
  }
};

module.exports = nextConfig;
