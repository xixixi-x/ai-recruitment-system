import React, {useEffect, useState} from 'react';
import {createRoot} from 'react-dom/client';
import './style.css';

const API = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api';
const EMPTY_PROFILE = {name: '', phone: '', email: '', education: '', school: '', experience: '', skills: ''};

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

function displayValue(value) {
  return value || '未填写';
}

async function request(path, options = {}) {
  const token = localStorage.getItem('candidate_token');
  const headers = options.headers || {};
  if (!(options.body instanceof FormData)) headers['Content-Type'] = 'application/json';
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(API + path, {...options, headers});
  const json = await res.json().catch(() => ({message: '响应解析失败'}));
  if (!res.ok || json.code !== 0) throw new Error(json.message || '请求失败');
  return json.data;
}

function Login({onLogin}) {
  const [mode, setMode] = useState('login');
  const [form, setForm] = useState({username: '', password: ''});
  const [err, setErr] = useState('');

  async function submit(e) {
    e.preventDefault();
    setErr('');
    try {
      if (mode === 'register') {
        await request('/auth/register', {method: 'POST', body: JSON.stringify({...form, role: 'candidate'})});
      }
      const data = await request('/auth/login', {method: 'POST', body: JSON.stringify({...form, role: 'candidate'})});
      localStorage.setItem('candidate_token', data.token);
      localStorage.setItem('candidate_user', JSON.stringify(data.user));
      onLogin(data.user);
    } catch (e) {
      setErr(e.message);
    }
  }

  return (
    <section className="auth">
      <p className="eyebrow">Candidate Access</p>
      <h2>候选人登录</h2>
      <form onSubmit={submit}>
        <input placeholder="账号" value={form.username} onChange={e => setForm({...form, username: e.target.value})} />
        <input
          type="password"
          placeholder="密码"
          value={form.password}
          onChange={e => setForm({...form, password: e.target.value})}
        />
        {err && <div className="error">{err}</div>}
        <button>{mode === 'login' ? '登录' : '注册并登录'}</button>
      </form>
      <button className="text-button" onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>
        {mode === 'login' ? '注册候选人账号' : '已有账号，返回登录'}
      </button>
    </section>
  );
}

function Jobs({user, onNeedLogin}) {
  const [jobs, setJobs] = useState([]);
  const [kw, setKw] = useState('');
  const [msg, setMsg] = useState('');

  async function load() {
    try {
      const data = await request('/jobs/public?keyword=' + encodeURIComponent(kw));
      setJobs(asArray(data?.items));
    } catch (e) {
      setMsg(e.message);
      setJobs([]);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function apply(id) {
    if (!user) {
      onNeedLogin();
      return;
    }
    try {
      await request(`/candidate/jobs/${id}/apply`, {method: 'POST'});
      setMsg('投递成功');
    } catch (e) {
      setMsg(e.message);
    }
  }

  return (
    <section className="panel">
      <div className="section-head">
        <div>
          <p className="eyebrow">Open Roles</p>
          <h2>公开岗位</h2>
        </div>
        <span className="count">{jobs.length} 个机会</span>
      </div>
      <div className="search">
        <input
          placeholder="搜索岗位或地点"
          value={kw}
          onChange={e => setKw(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') load(); }}
        />
        <button onClick={load}>搜索</button>
      </div>
      {msg && <div className="notice">{msg}</div>}
      <div className="jobs">
        {jobs.map(job => (
          <article className="job" key={job.id}>
            <div>
              <h3>{job.title}</h3>
              <p className="meta">{job.location || '地点待定'} · {job.salary || '薪资面议'}</p>
            </div>
            <p>{job.description || '暂无岗位描述'}</p>
            <details>
              <summary>岗位要求</summary>
              <div>{job.requirements || '暂无详细要求'}</div>
            </details>
            <button onClick={() => apply(job.id)}>一键投递</button>
          </article>
        ))}
      </div>
    </section>
  );
}

function Profile() {
  const [p, setP] = useState(EMPTY_PROFILE);
  const [msg, setMsg] = useState('');
  const [editing, setEditing] = useState(false);

  async function load() {
    try {
      const data = await request('/candidate/profile');
      setP(data && typeof data === 'object' ? data : EMPTY_PROFILE);
    } catch (e) {
      setMsg(e.message);
      setP(EMPTY_PROFILE);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function save() {
    try {
      const data = await request('/candidate/profile', {method: 'PUT', body: JSON.stringify(p)});
      setP(data);
      setMsg('资料已保存');
      setEditing(false);
    } catch (e) {
      setMsg(e.message);
    }
  }

  async function upload(e) {
    const file = e.target.files[0];
    if (!file) return;
    setMsg('正在获取签名 URL...');
    try {
      const sign = await request('/candidate/resume/sign-upload', {
        method: 'POST',
        body: JSON.stringify({filename: file.name, contentType: file.type, size: file.size}),
      });
      setMsg('正在上传简历...');
      const put = await fetch(sign.uploadUrl, {method: 'PUT', body: file});
      if (!put.ok) throw new Error('OSS 上传失败，请检查 Bucket CORS 与签名配置');
      const data = await request('/candidate/resume/confirm', {
        method: 'POST',
        body: JSON.stringify({objectKey: sign.objectKey, filename: file.name}),
      });
      setP(data);
      setMsg('简历已上传并记录');
    } catch (e) {
      setMsg(e.message);
    }
  }

  if (!editing) {
    return (
      <section className="panel">
        <div className="section-head">
          <div>
            <p className="eyebrow">Profile</p>
            <h2>结构化个人档案</h2>
          </div>
          <button className="ghost-button" onClick={() => setEditing(true)}>修改资料</button>
        </div>
        <div className="profile-view">
          <div><span>姓名</span><b>{displayValue(p.name)}</b></div>
          <div><span>电话</span><b>{displayValue(p.phone)}</b></div>
          <div><span>邮箱</span><b>{displayValue(p.email)}</b></div>
          <div><span>最高学历</span><b>{displayValue(p.education)}</b></div>
          <div><span>毕业院校</span><b>{displayValue(p.school)}</b></div>
          <div><span>核心技能</span><b>{displayValue(p.skills)}</b></div>
          <div className="wide"><span>工作 / 项目经历</span><b>{displayValue(p.experience)}</b></div>
          <div className="wide"><span>简历文件</span><b>{p.resumeFileName || '未上传'}</b></div>
        </div>
        {msg && <div className="notice">{msg}</div>}
      </section>
    );
  }

  return (
    <section className="panel">
      <div className="section-head">
        <div>
          <p className="eyebrow">Profile</p>
          <h2>编辑个人档案</h2>
        </div>
        <button className="ghost-button" onClick={() => setEditing(false)}>取消</button>
      </div>
      <div className="form-grid">
        <input placeholder="姓名" value={p.name || ''} onChange={e => setP({...p, name: e.target.value})} />
        <input placeholder="电话" value={p.phone || ''} onChange={e => setP({...p, phone: e.target.value})} />
        <input placeholder="邮箱" value={p.email || ''} onChange={e => setP({...p, email: e.target.value})} />
        <input placeholder="最高学历" value={p.education || ''} onChange={e => setP({...p, education: e.target.value})} />
        <input placeholder="毕业院校" value={p.school || ''} onChange={e => setP({...p, school: e.target.value})} />
        <input placeholder="核心技能标签" value={p.skills || ''} onChange={e => setP({...p, skills: e.target.value})} />
        <textarea placeholder="工作 / 项目经历" value={p.experience || ''} onChange={e => setP({...p, experience: e.target.value})} />
      </div>
      <div className="actions">
        <button onClick={save}>保存资料</button>
        <label className="upload">
          上传 PDF / DOC / DOCX 简历
          <input type="file" accept=".pdf,.doc,.docx" onChange={upload} />
        </label>
      </div>
      {p.resumeFileName && <p className="file-name">已上传简历：{p.resumeFileName}</p>}
      {msg && <div className="notice">{msg}</div>}
    </section>
  );
}

function MyApplications() {
  const [apps, setApps] = useState([]);

  useEffect(() => {
    request('/candidate/applications').then(data => setApps(asArray(data))).catch(() => setApps([]));
  }, []);

  return (
    <section className="panel">
      <div className="section-head">
        <div>
          <p className="eyebrow">Applications</p>
          <h2>我的投递</h2>
        </div>
        <span className="count">{apps.length} 条记录</span>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>岗位</th>
              <th>状态</th>
              <th>时间</th>
            </tr>
          </thead>
          <tbody>
            {apps.map(app => (
              <tr key={app.id}>
                <td>{app.jobTitle}</td>
                <td><span className="status">{app.status}</span></td>
                <td>{new Date(app.createdAt).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function App() {
  const [user, setUser] = useState(() => readStoredUser('candidate_user'));
  const [showLogin, setShowLogin] = useState(false);

  return (
    <div>
      <header className="topbar">
        <div>
          <p className="eyebrow">Recruitment System</p>
          <h1>候选人中心</h1>
        </div>
        <div className="user-area">
          {user ? (
            <>
              <span>{user.username}</span>
              <button className="ghost-button" onClick={() => { localStorage.clear(); setUser(null); }}>退出</button>
            </>
          ) : (
            <button onClick={() => setShowLogin(true)}>登录 / 注册</button>
          )}
        </div>
      </header>
      <main className="page">
        <section className="hero">
          <div>
            <h2>清楚管理求职资料与投递进度</h2>
            <p>浏览岗位、维护档案、上传简历，并持续跟进每一次申请。</p>
          </div>
          <div className="hero-tags">
            <span>岗位检索</span>
            <span>档案维护</span>
            <span>投递记录</span>
          </div>
        </section>
        {showLogin && !user && <Login onLogin={u => { setUser(u); setShowLogin(false); }} />}
        <Jobs user={user} onNeedLogin={() => setShowLogin(true)} />
        {user && (
          <>
            <Profile />
            <MyApplications />
          </>
        )}
      </main>
    </div>
  );
}

createRoot(document.getElementById('root')).render(<App />);
