import React, {useEffect, useState} from 'react';
import {createRoot} from 'react-dom/client';
import './style.css';

const API = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api';
const EMPTY_PROFILE = {name: '', phone: '', email: '', education: '', school: '', experience: '', skills: ''};
const EDUCATION_OPTIONS = ['初中', '高中', '专科', '本科', '研究生', '博士生'];

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

function validateProfile(profile) {
  if (!/^\d{11}$/.test(profile.phone || '')) return '电话必须是 11 位数字';
  if (!/^[^\s@]+@[^\s@]+\.com$/.test(profile.email || '')) return '邮箱格式必须类似 xx@xx.com';
  if (!EDUCATION_OPTIONS.includes(profile.education || '')) return '最高学历必须从可选项中选择';
  return '';
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

function Auth({onLogin}) {
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
    <main className="auth-page">
      <section className="auth-copy">
        <p className="eyebrow">Candidate Center</p>
        <h1>清楚管理求职资料与投递进度</h1>
        <p>维护档案、上传简历、浏览岗位，并持续跟进每一次申请。</p>
        <div className="auth-metrics">
          <span>岗位检索</span>
          <span>档案维护</span>
          <span>投递记录</span>
        </div>
        <div className="auth-preview">
          <div>
            <span>01</span>
            <b>完善档案</b>
            <p>补充学历、学校、项目经历和核心技能。</p>
          </div>
          <div>
            <span>02</span>
            <b>上传简历</b>
            <p>将 PDF/DOC/DOCX 简历安全保存到对象存储。</p>
          </div>
          <div>
            <span>03</span>
            <b>跟进投递</b>
            <p>统一查看岗位申请状态和投递时间。</p>
          </div>
        </div>
      </section>
      <section className="auth-card">
        <p className="eyebrow">{mode === 'login' ? 'Sign in' : 'Create account'}</p>
        <h2>{mode === 'login' ? '候选人登录' : '注册候选人账号'}</h2>
        <form onSubmit={submit}>
          <label>
            <span>账号</span>
            <input placeholder="请输入账号" value={form.username} onChange={e => setForm({...form, username: e.target.value})} />
          </label>
          <label>
            <span>密码</span>
            <input
              type="password"
              placeholder="请输入密码"
              value={form.password}
              onChange={e => setForm({...form, password: e.target.value})}
            />
          </label>
          {err && <div className="error">{err}</div>}
          <button>{mode === 'login' ? '进入候选人中心' : '注册并进入'}</button>
        </form>
        <button className="text-button" onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>
          {mode === 'login' ? '注册候选人账号' : '已有账号？返回登录'}
        </button>
      </section>
    </main>
  );
}

function Jobs({user}) {
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
            <button onClick={() => apply(job.id)} disabled={!user}>一键投递</button>
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
    const validationMessage = validateProfile(p);
    if (validationMessage) {
      setMsg(validationMessage);
      return;
    }

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
    setMsg('正在上传简历...');
    try {
      const form = new FormData();
      form.append('resume', file);
      const data = await request('/candidate/resume/upload', {
        method: 'POST',
        body: form,
      });
      setP(data);
      setMsg('简历已上传到 OSS 并记录');
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
        <input
          placeholder="电话，11 位数字"
          value={p.phone || ''}
          onChange={e => setP({...p, phone: e.target.value.replace(/\D/g, '').slice(0, 11)})}
        />
        <input placeholder="邮箱，xx@xx.com" value={p.email || ''} onChange={e => setP({...p, email: e.target.value})} />
        <select value={p.education || ''} onChange={e => setP({...p, education: e.target.value})}>
          <option value="">请选择最高学历</option>
          {EDUCATION_OPTIONS.map(item => <option key={item} value={item}>{item}</option>)}
        </select>
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

function Overview({profile, applicationCount}) {
  const completed = profile?.name && profile?.phone && profile?.email && profile?.education && profile?.school && profile?.skills;
  return (
    <section className="overview-grid">
      <div>
        <span>档案状态</span>
        <b>{completed ? '信息完整' : '待完善'}</b>
      </div>
      <div>
        <span>简历文件</span>
        <b>{profile?.resumeFileName ? '已上传' : '未上传'}</b>
      </div>
      <div>
        <span>投递记录</span>
        <b>{applicationCount} 条</b>
      </div>
    </section>
  );
}

function App() {
  const [user, setUser] = useState(() => readStoredUser('candidate_user'));
  const [profileSummary, setProfileSummary] = useState(EMPTY_PROFILE);
  const [applicationCount, setApplicationCount] = useState(0);

  useEffect(() => {
    if (!user) return;
    request('/candidate/profile')
      .then(data => setProfileSummary(data && typeof data === 'object' ? data : EMPTY_PROFILE))
      .catch(() => setProfileSummary(EMPTY_PROFILE));
    request('/candidate/applications')
      .then(data => setApplicationCount(asArray(data).length))
      .catch(() => setApplicationCount(0));
  }, [user]);

  if (!user) return <Auth onLogin={setUser} />;

  return (
    <div className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Recruitment System</p>
          <h1>候选人中心</h1>
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
            <h2>清楚管理求职资料与投递进度</h2>
            <p>浏览岗位、维护档案、上传简历，并持续跟进每一次申请。</p>
          </div>
          <div className="hero-tags">
            <span>岗位检索</span>
            <span>档案维护</span>
            <span>投递记录</span>
          </div>
        </section>
        <Overview profile={profileSummary} applicationCount={applicationCount} />
        <Profile />
        <Jobs user={user} />
        <MyApplications />
      </main>
    </div>
  );
}

createRoot(document.getElementById('root')).render(<App />);
