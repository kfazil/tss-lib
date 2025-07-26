module.exports = {
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'https://automatic-barnacle-qvvp65v66663xj9v-40715.app.github.dev/api/:path*', // Proxy to backend
        // destination: 'http://localhost:40715/api/:path*', // Proxy to backend
      },
    ];
  },
};
