import React, {useEffect, useState} from 'react';
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

async function request(path, options={}) {
  const token = localStorage.getItem('hr_token');
  const headers = options.headers || {};
  if (!(options.body instanceof FormData)) headers['Content-Type'] = 'application/json';
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(API + path, {...options, headers});
  const json = await res.json().catch(()=>({message:'响应解析失败'}));
  if (!res.ok || json.code !== 0) throw new Error(json.message || '请求失败');
  return json.data;
}

function Auth({onLogin}) {
  const [mode, setMode] = useState('login');
  const [form, setForm] = useState({username:'', password:''});
  const [err, setErr] = useState('');
  async function submit(e) {
    e.preventDefault(); setErr('');
    try {
      if (mode === 'register') await request('/auth/register', {method:'POST', body: JSON.stringify({...form, role:'hr'})});
      const data = await request('/auth/login', {method:'POST', body: JSON.stringify({...form, role:'hr'})});
      localStorage.setItem('hr_token', data.token); localStorage.setItem('hr_user', JSON.stringify(data.user)); onLogin(data.user);
    } catch(e) { setErr(e.message); }
  }
  return <div className="auth-card">
    <h1>HR 管理端</h1><p>独立账号密码登录，JWT 令牌访问后台功能</p>
    <form onSubmit={submit}>
      <input placeholder="HR账号" value={form.username} onChange={e=>setForm({...form, username:e.target.value})}/>
      <input placeholder="密码至少6位" type="password" value={form.password} onChange={e=>setForm({...form, password:e.target.value})}/>
      {err && <div className="error">{err}</div>}
      <button>{mode === 'login' ? '登录' : '注册并登录'}</button>
    </form>
    <button className="link" onClick={()=>setMode(mode==='login'?'register':'login')}>{mode==='login'?'没有账号？注册 HR':'已有账号？登录'}</button>
  </div>
}

function JobPanel() {
  const [jobs, setJobs] = useState([]);
  const [form, setForm] = useState({title:'', description:'', requirements:'', salary:'', location:''});
  const [msg, setMsg] = useState('');
  async function load(){
    try {
      setJobs(asArray(await request('/hr/jobs')));
    } catch(e) {
      setMsg(e.message);
      setJobs([]);
    }
  }
  useEffect(()=>{load()},[]);
  async function createJob(e){ e.preventDefault(); setMsg(''); try { await request('/hr/jobs',{method:'POST', body:JSON.stringify(form)}); setForm({title:'',description:'',requirements:'',salary:'',location:''}); setMsg('岗位已发布'); load(); } catch(e){setMsg(e.message)} }
  return <section className="panel"><h2>岗位管理</h2>
    <form className="grid" onSubmit={createJob}>
      <input placeholder="岗位名称" value={form.title} onChange={e=>setForm({...form,title:e.target.value})}/>
      <input placeholder="薪资" value={form.salary} onChange={e=>setForm({...form,salary:e.target.value})}/>
      <input placeholder="地点" value={form.location} onChange={e=>setForm({...form,location:e.target.value})}/>
      <textarea placeholder="岗位描述" value={form.description} onChange={e=>setForm({...form,description:e.target.value})}/>
      <textarea placeholder="岗位要求" value={form.requirements} onChange={e=>setForm({...form,requirements:e.target.value})}/>
      <button>发布岗位</button>
    </form>{msg && <div className="notice">{msg}</div>}
    <div className="cards">{jobs.map(j=><div className="card" key={j.id}><b>{j.title}</b><span>{j.location} · {j.salary}</span><p>{j.description}</p></div>)}</div>
  </section>
}

function ApplicationPanel() {
  const [apps, setApps] = useState([]); const [err,setErr]=useState('');
  async function load(){
    try {
      setApps(asArray(await request('/hr/applications')));
    } catch(e) {
      setErr(e.message);
      setApps([]);
    }
  }
  useEffect(()=>{load()},[]);
  async function openResume(id){ try{ const data = await request(`/hr/applications/${id}/resume-url`); window.open(data.url, '_blank'); }catch(e){alert(e.message)} }
  return <section className="panel"><h2>候选人投递</h2>{err&&<div className="error">{err}</div>}
    <table><thead><tr><th>岗位</th><th>候选人</th><th>联系方式</th><th>学历/学校</th><th>技能</th><th>简历</th></tr></thead><tbody>{apps.map(a=><tr key={a.id}><td>{a.jobTitle}</td><td>{a.candidateName||'未填写'}</td><td>{a.phone}<br/>{a.email}</td><td>{a.education}<br/>{a.school}</td><td>{a.skills}</td><td><button onClick={()=>openResume(a.id)}>签名下载</button></td></tr>)}</tbody></table>
  </section>
}

function ChatPanel(){
  const [q,setQ]=useState(''); const [msgs,setMsgs]=useState([]); const [loading,setLoading]=useState(false);
  async function load(){
    try {
      setMsgs(asArray(await request('/hr/ai/history')));
    } catch {
      setMsgs([]);
    }
  }
  useEffect(()=>{load()},[]);
  async function send(){ if(!q.trim())return; setLoading(true); const question=q; setQ(''); setMsgs(m=>[...m,{role:'user',content:question}]); try{ const data=await request('/hr/ai/chat',{method:'POST', body:JSON.stringify({question})}); setMsgs(m=>[...m,{role:'assistant',content:data.answer}]); }catch(e){setMsgs(m=>[...m,{role:'assistant',content:'错误：'+e.message}]);} finally{setLoading(false);} }
  return <section className="panel chat"><h2>AI 招聘问答</h2><div className="chat-box">{msgs.map((m,i)=><div key={i} className={'msg '+m.role}>{m.content}</div>)}</div><div className="chat-input"><input value={q} onChange={e=>setQ(e.target.value)} placeholder="例如：哪个岗位投递最多？" onKeyDown={e=>{if(e.key==='Enter')send()}}/><button disabled={loading} onClick={send}>{loading?'思考中':'发送'}</button></div></section>
}

function App(){
  const [user,setUser]=useState(()=>readStoredUser('hr_user'));
  if(!user) return <Auth onLogin={setUser}/>;
  return <div><header><h1>智能招聘系统 · HR端</h1><div>{user.username}<button onClick={()=>{localStorage.clear();setUser(null)}}>退出</button></div></header><main><JobPanel/><ApplicationPanel/><ChatPanel/></main></div>
}

createRoot(document.getElementById('root')).render(<App/>);
