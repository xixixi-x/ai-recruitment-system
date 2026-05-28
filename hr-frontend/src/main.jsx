import React, {useEffect, useMemo, useState} from 'react';
import {createRoot} from 'react-dom/client';
import './style.css';

const API = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api';

function readStoredUser(key) {
  try {
    return JSON.parse(localStorage.getItem(key) || 'null');
  } catch {
    localStorage.removeItem(key);
    return null;
  }
}

function asArray(value) {
  return Array.isArray(value) ? value : [];
}

async function request(path, options = {}) {
  const token = localStorage.getItem('hr_token');
  const headers = options.headers || {};
  if (!(options.body instanceof FormData)) headers['Content-Type'] = 'application/json';
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(API + path, {...options, headers});
  const json = await res.json().catch(() => ({message: '响应解析失败'}));
  if (!res.ok || json.code !== 0) throw new Error(json.message || '请求失败');
  return json.data;
}

function Auth({onLogin}) {
  const [mode, setMode] = useState('login');
  const [form, setForm] = useState({username: '', password: ''});
  const [err, setErr] = useState('');

  async function submit(e) {
    e.preventDefault();
    setErr('');
    try {
      if (mode === 'register') {
        await request('/auth/register', {method: 'POST', body: JSON.stringify({...form, role: 'hr'})});
      }
      const data = await request('/auth/login', {method: 'POST', body: JSON.stringify({...form, role: 'hr'})});
      localStorage.setItem('hr_token', data.token);
      localStorage.setItem('hr_user', JSON.stringify(data.user));
      onLogin(data.user);
    } catch (e) {
      setErr(e.message);
    }
  }

  return (
    <main className="auth-page">
      <section className="auth-copy">
        <p className="eyebrow">HR Workspace</p>
        <h1>智能招聘管理台</h1>
        <p>集中管理岗位、候选人投递和 AI 招聘问答，让招聘流程保持清晰、有序、可追踪。</p>
        <div className="auth-metrics">
          <span>岗位发布</span>
          <span>投递筛选</span>
          <span>AI 分析</span>
        </div>
        <div className="auth-preview">
          <div>
            <span>01</span>
            <b>发布岗位</b>
            <p>录入职责、地点、薪资和技能要求。</p>
          </div>
          <div>
            <span>02</span>
            <b>查看投递</b>
            <p>集中比较候选人资料、简历和联系方式。</p>
          </div>
          <div>
            <span>03</span>
            <b>AI 辅助</b>
            <p>快速询问岗位热度和候选人分布。</p>
          </div>
        </div>
      </section>
      <section className="auth-card">
        <p className="eyebrow">{mode === 'login' ? 'Sign in' : 'Create account'}</p>
        <h2>{mode === 'login' ? 'HR 登录' : '注册 HR 账号'}</h2>
        <form onSubmit={submit}>
          <label>
            <span>账号</span>
            <input placeholder="请输入 HR 账号" value={form.username} onChange={e => setForm({...form, username: e.target.value})} />
          </label>
          <label>
            <span>密码</span>
            <input
              type="password"
              placeholder="密码至少 6 位"
              value={form.password}
              onChange={e => setForm({...form, password: e.target.value})}
            />
          </label>
          {err && <div className="error">{err}</div>}
          <button>{mode === 'login' ? '登录工作台' : '注册并进入'}</button>
        </form>
        <button className="text-button" onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>
          {mode === 'login' ? '没有账号？注册 HR' : '已有账号？返回登录'}
        </button>
      </section>
    </main>
  );
}

function JobPanel() {
  const [jobs, setJobs] = useState([]);
  const [form, setForm] = useState({title: '', description: '', requirements: '', salary: '', location: ''});
  const [msg, setMsg] = useState('');
  const [creating, setCreating] = useState(false);

  async function load() {
    try {
      setJobs(asArray(await request('/hr/jobs')));
    } catch (e) {
      setMsg(e.message);
      setJobs([]);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function createJob(e) {
    e.preventDefault();
    setMsg('');
    try {
      await request('/hr/jobs', {method: 'POST', body: JSON.stringify(form)});
      setForm({title: '', description: '', requirements: '', salary: '', location: ''});
      setMsg('岗位已发布');
      setCreating(false);
      load();
    } catch (e) {
      setMsg(e.message);
    }
  }

  return (
    <section className="panel">
      <div className="section-head">
        <div>
          <p className="eyebrow">Jobs</p>
          <h2>岗位管理</h2>
        </div>
        <div className="section-actions">
          <span className="count">{jobs.length} 个岗位</span>
          <button className="ghost-button" onClick={() => setCreating(v => !v)}>{creating ? '收起发布' : '新建岗位'}</button>
        </div>
      </div>
      {creating && (
        <form className="form-grid create-job" onSubmit={createJob}>
          <input placeholder="岗位名称" value={form.title} onChange={e => setForm({...form, title: e.target.value})} />
          <input placeholder="薪资范围" value={form.salary} onChange={e => setForm({...form, salary: e.target.value})} />
          <input placeholder="工作地点" value={form.location} onChange={e => setForm({...form, location: e.target.value})} />
          <textarea placeholder="岗位描述" value={form.description} onChange={e => setForm({...form, description: e.target.value})} />
          <textarea placeholder="岗位技能要求" value={form.requirements} onChange={e => setForm({...form, requirements: e.target.value})} />
          <button className="form-submit">发布岗位</button>
        </form>
      )}
      {msg && <div className="notice">{msg}</div>}
      <div className="job-list">
        {jobs.map(job => (
          <article className="job-card" key={job.id}>
            <div>
              <h3>{job.title}</h3>
              <p className="meta">{job.location || '地点待定'} · {job.salary || '薪资面议'}</p>
            </div>
            <p>{job.description || '暂无岗位描述'}</p>
            <details className="job-requirements">
              <summary>岗位技能要求</summary>
              <div>{job.requirements || '暂无技能要求'}</div>
            </details>
          </article>
        ))}
      </div>
    </section>
  );
}

function ApplicationPanel() {
  const [apps, setApps] = useState([]);
  const [err, setErr] = useState('');

  async function load() {
    try {
      setApps(asArray(await request('/hr/applications')));
    } catch (e) {
      setErr(e.message);
      setApps([]);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function openResume(id) {
    try {
      const data = await request(`/hr/applications/${id}/resume-url`);
      window.open(data.url, '_blank');
    } catch (e) {
      alert(e.message);
    }
  }

  return (
    <section className="panel">
      <div className="section-head">
        <div>
          <p className="eyebrow">Applications</p>
          <h2>候选人投递</h2>
        </div>
        <span className="count">{apps.length} 份投递</span>
      </div>
      {err && <div className="error">{err}</div>}
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>岗位</th>
              <th>候选人</th>
              <th>电话</th>
              <th>邮箱</th>
              <th>学校 / 学历</th>
              <th>技能</th>
              <th>简历</th>
            </tr>
          </thead>
          <tbody>
            {apps.map(app => (
              <tr key={app.id}>
                <td>{app.jobTitle}</td>
                <td>{app.candidateName || '未填写'}</td>
                <td>{app.phone || '未填写'}</td>
                <td>{app.email || '未填写'}</td>
                <td>{app.school || '未填写'} / {app.education || '未填写'}</td>
                <td>{app.skills || '未填写'}</td>
                <td><button className="ghost-button" onClick={() => openResume(app.id)}>下载</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function ChatPanel() {
  const [q, setQ] = useState('');
  const [msgs, setMsgs] = useState([]);
  const [loading, setLoading] = useState(false);

  async function load() {
    try {
      setMsgs(asArray(await request('/hr/ai/history')));
    } catch {
      setMsgs([]);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function send() {
    if (!q.trim()) return;
    setLoading(true);
    const question = q;
    setQ('');
    setMsgs(m => [...m, {role: 'user', content: question}]);
    try {
      const data = await request('/hr/ai/chat', {method: 'POST', body: JSON.stringify({question})});
      setMsgs(m => [...m, {role: 'assistant', content: data.answer}]);
    } catch (e) {
      setMsgs(m => [...m, {role: 'assistant', content: '错误：' + e.message}]);
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="panel chat">
      <div className="section-head">
        <div>
          <p className="eyebrow">Assistant</p>
          <h2>AI 招聘问答</h2>
        </div>
      </div>
      <div className="chat-box">
        {msgs.length === 0 && <p className="empty">还没有对话记录。</p>}
        {msgs.map((m, i) => <div key={i} className={'msg ' + m.role}>{m.content}</div>)}
      </div>
      <div className="chat-input">
        <input
          value={q}
          onChange={e => setQ(e.target.value)}
          placeholder="例如：哪个岗位投递最多？"
          onKeyDown={e => { if (e.key === 'Enter') send(); }}
        />
        <button disabled={loading} onClick={send}>{loading ? '思考中' : '发送'}</button>
      </div>
    </section>
  );
}

function App() {
  const [user, setUser] = useState(() => readStoredUser('hr_user'));
  const overview = useMemo(() => ['岗位发布', '投递管理', '简历下载', 'AI 分析'], []);

  if (!user) return <Auth onLogin={setUser} />;

  return (
    <div className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Recruitment System</p>
          <h1>HR 工作台</h1>
        </div>
        <div className="user-area">
          <span>{user.username}</span>
          <button className="ghost-button" onClick={() => { localStorage.clear(); setUser(null); }}>退出</button>
        </div>
      </header>
      <main className="page">
        <section className="hero">
          <div>
            <p className="eyebrow">Overview</p>
            <h2>把招聘流程收拢到一个清晰界面</h2>
            <p>发布岗位、筛选候选人、查看简历和提问 AI 都保持在同一套节奏里。</p>
          </div>
          <div className="hero-tags">
            {overview.map(item => <span key={item}>{item}</span>)}
          </div>
        </section>
        <JobPanel />
        <ApplicationPanel />
        <ChatPanel />
      </main>
    </div>
  );
}

createRoot(document.getElementById('root')).render(<App />);
