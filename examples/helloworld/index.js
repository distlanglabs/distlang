export default {
  async fetch(request, env, ctx) {
    console.log("Hello worker fetch will return a Response object");
    return new Response("Hello Worker!");
  },
};
