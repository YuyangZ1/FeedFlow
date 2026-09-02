import React, { useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  ImagePlus,
  LogIn,
  LogOut,
  RefreshCw,
  Search,
  Send,
  Sparkles,
  Trash2,
  UserPlus
} from "lucide-react";
import "./styles.css";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

function decodeUsername(token) {
  try {
    const base64Url = token.split(".")[1] || "";
    const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), "=");
    const payload = JSON.parse(atob(padded));
    return payload.username || "";
  } catch {
    return "";
  }
}

async function request(path, options = {}) {
  const headers = options.headers ? { ...options.headers } : {};
  const token = localStorage.getItem("feedflow_token");

  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  if (options.body && !(options.body instanceof FormData)) {
    headers["Content-Type"] = "application/json";
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers
  });
  const payload = await response.json().catch(() => ({}));

  if (!response.ok) {
    throw new Error(payload.error || "Request failed");
  }
  return payload;
}

function App() {
  const [token, setToken] = useState(() => localStorage.getItem("feedflow_token") || "");
  const username = useMemo(() => decodeUsername(token), [token]);

  function handleToken(nextToken) {
    localStorage.setItem("feedflow_token", nextToken);
    setToken(nextToken);
  }

  function logout() {
    localStorage.removeItem("feedflow_token");
    setToken("");
  }

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">FeedFlow</p>
          <h1>Compose, publish, and search posts</h1>
        </div>
        {token && (
          <button className="ghost iconText" onClick={logout} type="button">
            <LogOut size={18} />
            <span>{username || "Sign out"}</span>
          </button>
        )}
      </header>

      {!token ? (
        <AuthPanel onToken={handleToken} />
      ) : (
        <Dashboard username={username} />
      )}
    </main>
  );
}

function AuthPanel({ onToken }) {
  const [mode, setMode] = useState("signin");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [age, setAge] = useState("");
  const [gender, setGender] = useState("");
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event) {
    event.preventDefault();
    setBusy(true);
    setStatus("");

    try {
      if (mode === "signup") {
        await request("/signup", {
          method: "POST",
          body: JSON.stringify({
            username,
            password,
            age: Number(age) || 0,
            gender
          })
        });
      }

      const payload = await request("/signin", {
        method: "POST",
        body: JSON.stringify({ username, password })
      });
      onToken(payload.token);
    } catch (error) {
      setStatus(error.message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="authLayout">
      <form className="panel authPanel" onSubmit={submit}>
        <div className="segmented" aria-label="Authentication mode">
          <button
            className={mode === "signin" ? "active" : ""}
            onClick={() => setMode("signin")}
            type="button"
          >
            <LogIn size={17} />
            Sign in
          </button>
          <button
            className={mode === "signup" ? "active" : ""}
            onClick={() => setMode("signup")}
            type="button"
          >
            <UserPlus size={17} />
            Sign up
          </button>
        </div>

        <label>
          Username
          <input
            autoComplete="username"
            onChange={(event) => setUsername(event.target.value)}
            placeholder="alice_chen"
            required
            value={username}
          />
        </label>
        <label>
          Password
          <input
            autoComplete={mode === "signup" ? "new-password" : "current-password"}
            minLength={8}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="At least 8 characters"
            required
            type="password"
            value={password}
          />
        </label>

        {mode === "signup" && (
          <div className="inlineFields">
            <label>
              Age
              <input
                min="0"
                onChange={(event) => setAge(event.target.value)}
                type="number"
                value={age}
              />
            </label>
            <label>
              Gender
              <input onChange={(event) => setGender(event.target.value)} value={gender} />
            </label>
          </div>
        )}

        {status && <p className="alert">{status}</p>}
        <button className="primary iconText" disabled={busy} type="submit">
          {mode === "signin" ? <LogIn size={18} /> : <UserPlus size={18} />}
          {busy ? "Working..." : mode === "signin" ? "Sign in" : "Create account"}
        </button>
      </form>
    </section>
  );
}

function Dashboard({ username }) {
  const [refreshKey, setRefreshKey] = useState(0);

  return (
    <div className="dashboard">
      <Composer onPublished={() => setRefreshKey((value) => value + 1)} />
      <Feed refreshKey={refreshKey} username={username} />
    </div>
  );
}

function Composer({ onPublished }) {
  const [message, setMessage] = useState("");
  const [prompt, setPrompt] = useState("");
  const [file, setFile] = useState(null);
  const [status, setStatus] = useState("");
  const [generating, setGenerating] = useState(false);
  const [uploading, setUploading] = useState(false);

  async function generatePost() {
    setGenerating(true);
    setStatus("");

    try {
      const payload = await request("/ai/post", {
        method: "POST",
        body: JSON.stringify({ prompt })
      });
      setMessage(payload.text);
    } catch (error) {
      setStatus(error.message);
    } finally {
      setGenerating(false);
    }
  }

  async function publish(event) {
    event.preventDefault();
    if (!file) {
      setStatus("Choose an image or video first");
      return;
    }

    const form = new FormData();
    form.append("message", message);
    form.append("media_file", file);

    setUploading(true);
    setStatus("");
    try {
      await request("/upload", {
        method: "POST",
        body: form
      });
      setMessage("");
      setPrompt("");
      setFile(null);
      event.target.reset();
      setStatus("Post published");
      onPublished();
    } catch (error) {
      setStatus(error.message);
    } finally {
      setUploading(false);
    }
  }

  return (
    <section className="panel composer">
      <div className="panelHeader">
        <div>
          <p className="eyebrow">Create</p>
          <h2>New post</h2>
        </div>
        <Sparkles size={21} />
      </div>

      <label>
        AI prompt
        <div className="actionRow">
          <input
            onChange={(event) => setPrompt(event.target.value)}
            placeholder="Weekend hike recap with a warm tone"
            value={prompt}
          />
          <button
            aria-label="Generate post text"
            className="iconButton"
            disabled={generating || !prompt.trim()}
            onClick={generatePost}
            title="Generate post text"
            type="button"
          >
            <Sparkles size={19} />
          </button>
        </div>
      </label>

      <form onSubmit={publish}>
        <label>
          Post text
          <textarea
            maxLength={700}
            onChange={(event) => setMessage(event.target.value)}
            placeholder="Write a post or generate one from a prompt"
            required
            value={message}
          />
        </label>

        <label className="fileInput">
          <ImagePlus size={18} />
          <span>{file ? file.name : "Choose media"}</span>
          <input
            accept="image/*,video/*"
            onChange={(event) => setFile(event.target.files?.[0] || null)}
            type="file"
          />
        </label>

        {status && <p className={status === "Post published" ? "notice" : "alert"}>{status}</p>}
        <button className="primary iconText" disabled={uploading} type="submit">
          <Send size={18} />
          {uploading ? "Publishing..." : "Publish"}
        </button>
      </form>
    </section>
  );
}

function Feed({ refreshKey, username }) {
  const [posts, setPosts] = useState([]);
  const [keywords, setKeywords] = useState("");
  const [userFilter, setUserFilter] = useState("");
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(false);
  const pageSize = 10;

  async function loadPosts(nextPage = page) {
    setLoading(true);
    setStatus("");

    const params = new URLSearchParams({
      page: String(nextPage),
      pageSize: String(pageSize)
    });
    if (keywords.trim()) {
      params.set("keywords", keywords.trim());
    }
    if (userFilter.trim()) {
      params.set("user", userFilter.trim());
    }

    try {
      const payload = await request(`/search?${params.toString()}`);
      setPosts(payload.posts || []);
      setTotal(payload.total || 0);
      setPage(payload.page || nextPage);
    } catch (error) {
      setStatus(error.message);
    } finally {
      setLoading(false);
    }
  }

  async function deletePost(id) {
    setStatus("");
    try {
      await request(`/post/${id}`, { method: "DELETE" });
      await loadPosts(page);
    } catch (error) {
      setStatus(error.message);
    }
  }

  useEffect(() => {
    loadPosts(1);
  }, [refreshKey]);

  const pageCount = Math.max(1, Math.ceil(total / pageSize));

  return (
    <section className="panel feed">
      <div className="panelHeader">
        <div>
          <p className="eyebrow">Browse</p>
          <h2>Posts</h2>
        </div>
        <button
          aria-label="Refresh posts"
          className="iconButton"
          disabled={loading}
          onClick={() => loadPosts(page)}
          title="Refresh posts"
          type="button"
        >
          <RefreshCw size={18} />
        </button>
      </div>

      <form
        className="filters"
        onSubmit={(event) => {
          event.preventDefault();
          loadPosts(1);
        }}
      >
        <label>
          Keywords
          <input
            onChange={(event) => setKeywords(event.target.value)}
            placeholder="Search post text"
            value={keywords}
          />
        </label>
        <label>
          User
          <input
            onChange={(event) => setUserFilter(event.target.value)}
            placeholder="Filter by username"
            value={userFilter}
          />
        </label>
        <button aria-label="Search posts" className="iconButton searchButton" type="submit">
          <Search size={19} />
        </button>
      </form>

      {status && <p className="alert">{status}</p>}
      {loading && <p className="muted">Loading posts...</p>}

      <div className="postList">
        {!loading && posts.length === 0 && <p className="empty">No posts found.</p>}
        {posts.map((post) => (
          <article className="postCard" key={post.id}>
            <div className="postMeta">
              <strong>@{post.user}</strong>
              {post.user === username && (
                <button
                  aria-label="Delete post"
                  className="iconButton danger"
                  onClick={() => deletePost(post.id)}
                  title="Delete post"
                  type="button"
                >
                  <Trash2 size={17} />
                </button>
              )}
            </div>
            <p>{post.message}</p>
            {post.type === "video" ? (
              <video controls src={post.url} />
            ) : (
              <img alt={post.message || "Post media"} src={post.url} />
            )}
          </article>
        ))}
      </div>

      <div className="pager">
        <button disabled={page <= 1 || loading} onClick={() => loadPosts(page - 1)} type="button">
          Previous
        </button>
        <span>
          {page} / {pageCount}
        </span>
        <button
          disabled={page >= pageCount || loading}
          onClick={() => loadPosts(page + 1)}
          type="button"
        >
          Next
        </button>
      </div>
    </section>
  );
}

createRoot(document.getElementById("root")).render(<App />);
