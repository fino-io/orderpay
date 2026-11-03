import { Link } from "react-router-dom";

export default function HomePage() {
  return (
    <main className="home-shell" aria-labelledby="home-title">
      <h1 id="home-title">Project workspace</h1>
      <p className="home-copy">Browse API operations and track project issues in one place.</p>
      <nav className="home-actions" aria-label="Quick links">
        <Link className="home-link" to="/console">Open API Console</Link>
        <Link className="home-link" to="/issues">Open Issues</Link>
      </nav>
    </main>
  );
}
