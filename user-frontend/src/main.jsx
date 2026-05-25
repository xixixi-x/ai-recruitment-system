import React, {useEffect, useState} from 'react';
import {createRoot} from 'react-dom/client';
import './style.css';

const API = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api';

const EMPTY_PROFILE = {name:'',phone:'',email:'',education:'',school:'',experience:'',skills:''};

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

async function request(path, options={}){
  const token=localStorage.getItem('candidate_token'); const headers=options.headers||{};
  if(!(options.body instanceof FormData)) headers['Content-Type']='application/json';
  if(token) headers.Authorization=`Bearer ${token}`;
  const res=await fetch(API+path,{...options,headers}); const json=await res.json().catch(()=>({message:'响应解析失败'}));
  if(!res.ok||json.code!==0) throw new Error(json.message||'请求失败'); return json.data;
}

function Login({onLogin}){
  const [mode,setMode]=useState('login'); const [form,setForm]=useState({username:'',password:''}); const [err,setErr]=useState('');
  async function submit(e){e.preventDefault();setErr('');try{if(mode==='register') await request('/auth/register',{method:'POST',body:JSON.stringify({...form,role:'candidate'})}); const data=await request('/auth/login',{method:'POST',body:JSON.stringify({...form,role:'candidate'})}); localStorage.setItem('candidate_token',data.token); localStorage.setItem('candidate_user',JSON.stringify(data.user)); onLogin(data.user);}catch(e){setErr(e.message)}}
  return <div className="auth"><h2>候选人登录</h2><form onSubmit={submit}><input placeholder="账号" value={form.username} onChange={e=>setForm({...form,username:e.target.value})}/><input type="password" placeholder="密码" value={form.password} onChange={e=>setForm({...form,password:e.target.value})}/>{err&&<div className="error">{err}</div>}<button>{mode==='login'?'登录':'注册并登录'}</button></form><button className="link" onClick={()=>setMode(mode==='login'?'register':'login')}>{mode==='login'?'注册候选人账号':'已有账号登录'}</button></div>
}

function Jobs({user,onNeedLogin}){
  const [jobs,setJobs]=useState([]); const [kw,setKw]=useState(''); const [msg,setMsg]=useState('');
  async function load(){
    try {
      const data = await request('/jobs/public?keyword=' + encodeURIComponent(kw));
      setJobs(asArray(data?.items));
    } catch(e) {
      setMsg(e.message);
      setJobs([]);
    }
  }
  useEffect(()=>{load()},[]);
  async function apply(id){ if(!user){onNeedLogin();return} try{await request(`/candidate/jobs/${id}/apply`,{method:'POST'}); setMsg('投递成功')}catch(e){setMsg(e.message)} }
  return <section className="panel"><h2>公开岗位列表</h2><div className="search"><input placeholder="搜索岗位/地点" value={kw} onChange={e=>setKw(e.target.value)}/><button onClick={load}>搜索</button></div>{msg&&<div className="notice">{msg}</div>}<div className="jobs">{jobs.map(j=><div className="job" key={j.id}><h3>{j.title}</h3><p>{j.location} · {j.salary||'薪资面议'}</p><p>{j.description}</p><details><summary>岗位要求</summary>{j.requirements}</details><button onClick={()=>apply(j.id)}>一键投递</button></div>)}</div></section>
}

function Profile(){
  const [p,setP]=useState(EMPTY_PROFILE); const [msg,setMsg]=useState('');
  async function load(){
    try {
      const data = await request('/candidate/profile');
      setP(data && typeof data === 'object' ? data : EMPTY_PROFILE);
    } catch(e) {
      setMsg(e.message);
      setP(EMPTY_PROFILE);
    }
  }
  useEffect(()=>{load()},[]);
  async function save(){try{const data=await request('/candidate/profile',{method:'PUT',body:JSON.stringify(p)}); setP(data); setMsg('资料已保存')}catch(e){setMsg(e.message)}}
  async function upload(e){const file=e.target.files[0]; if(!file)return; setMsg('正在获取签名URL...'); try{const sign=await request('/candidate/resume/sign-upload',{method:'POST',body:JSON.stringify({filename:file.name,contentType:file.type,size:file.size})}); setMsg('正在直传私有OSS...'); const put=await fetch(sign.uploadUrl,{method:'PUT',body:file}); if(!put.ok) throw new Error('OSS 上传失败，请检查 Bucket CORS 与签名配置'); const data=await request('/candidate/resume/confirm',{method:'POST',body:JSON.stringify({objectKey:sign.objectKey,filename:file.name})}); setP(data); setMsg('简历已上传到私有 OSS，并记录 objectKey');}catch(e){setMsg(e.message)}}
  return <section className="panel"><h2>结构化个人档案</h2><div className="grid"><input placeholder="姓名" value={p.name||''} onChange={e=>setP({...p,name:e.target.value})}/><input placeholder="电话" value={p.phone||''} onChange={e=>setP({...p,phone:e.target.value})}/><input placeholder="邮箱" value={p.email||''} onChange={e=>setP({...p,email:e.target.value})}/><input placeholder="最高学历" value={p.education||''} onChange={e=>setP({...p,education:e.target.value})}/><input placeholder="毕业院校" value={p.school||''} onChange={e=>setP({...p,school:e.target.value})}/><input placeholder="核心技能标签" value={p.skills||''} onChange={e=>setP({...p,skills:e.target.value})}/><textarea placeholder="工作/项目经历" value={p.experience||''} onChange={e=>setP({...p,experience:e.target.value})}/></div><div className="actions"><button onClick={save}>保存资料</button><label className="upload">上传PDF/DOC/DOCX简历<input type="file" accept=".pdf,.doc,.docx" onChange={upload}/></label></div>{p.resumeFileName&&<p>已上传简历：{p.resumeFileName}</p>}{msg&&<div className="notice">{msg}</div>}</section>
}

function MyApplications(){const [apps,setApps]=useState([]);useEffect(()=>{request('/candidate/applications').then(data=>setApps(asArray(data))).catch(()=>setApps([]))},[]);return <section className="panel"><h2>我的投递</h2><table><thead><tr><th>岗位</th><th>状态</th><th>时间</th></tr></thead><tbody>{apps.map(a=><tr key={a.id}><td>{a.jobTitle}</td><td>{a.status}</td><td>{new Date(a.createdAt).toLocaleString()}</td></tr>)}</tbody></table></section>}

function App(){const [user,setUser]=useState(()=>readStoredUser('candidate_user')); const [showLogin,setShowLogin]=useState(false);return <div><header><h1>智能招聘系统 · 候选人端</h1><div>{user? <><span>{user.username}</span><button onClick={()=>{localStorage.clear();setUser(null)}}>退出</button></> : <button onClick={()=>setShowLogin(true)}>登录/注册</button>}</div></header><main>{showLogin&&!user&&<Login onLogin={(u)=>{setUser(u);setShowLogin(false)}}/>}<Jobs user={user} onNeedLogin={()=>setShowLogin(true)}/>{user&&<><Profile/><MyApplications/></>}</main></div>}
createRoot(document.getElementById('root')).render(<App/>);
