const stage = process.env.NODE_ENV || "dev"
const isProduction = stage === "production"

export default {
  url: isProduction ? "https://devan.gg" : "http://localhost:4321",
  basePath:  isProduction ? "/brighten" : "/",
  github: "https://github.com/imdevan/brighten/",
  githubDocs: "https://github.com/imdevan/brighten/",
  title: "brighten",
  description: "Brghten (or darken) colors. Set the brightest color to white and scale remaining colors.",
}
