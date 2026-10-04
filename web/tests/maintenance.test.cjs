const fs = require('fs');
const vm = require('vm');
const assert = require('assert');
const script = fs.readFileSync('web/static/js/maintenance.js','utf8');
const tests=[];
function environment(){
 const timers=[];const storage=new Map();
 const ctx=vm.createContext({console,AbortController,Date,Math,localStorage:{getItem:k=>storage.get(k),setItem:(k,v)=>storage.set(k,v)},document:{getElementById:()=>null},fetch:async()=>{throw Error('not configured')},setTimeout:(fn,delay)=>{timers.push({fn,delay});return timers.length},clearTimeout:()=>{},globalThis:{}});
 vm.runInContext(script,ctx);return {ctx,timers,storage};
}
function button(){return {isConnected:true,disabled:false,dataset:{action:'update'},classList:{add(){},remove(){}},remove(){this.removed=true}}}
function json(data,status=200){return {ok:status<400,text:async()=>JSON.stringify(data)}}
async function test(name,fn){await fn();tests.push(name);console.log('PASS '+name)}
(async()=>{
 await test('Polling disconnect keeps button disabled and schedules status check',async()=>{const {ctx,timers}=environment();ctx.fetch=async()=>{throw Error('proxy disconnected')};const b=button(),s={};await ctx.waitForUpdate('caddy',{id:'job12345',started:Date.now()},b,s,0);assert(b.disabled);assert(s.textContent.includes('checking result'));assert(timers.some(t=>t.delay===2000));assert.notEqual(b.textContent,'Retry')});
 await test('Lost POST response is tracked using client job identifier',async()=>{const {ctx,storage}=environment();let post=0;ctx.fetch=async(_url,opts)=>{if(opts.method==='POST'){post++;throw Error('lost response')};return json({status:'running'})};const b=button(),s={};await ctx.startUpdate('caddy',b,s,0);await new Promise(setImmediate);assert.equal(post,1);assert(b.disabled);const pending=JSON.parse(storage.get('dock-pending-updates'));assert(pending.caddy.id)});
 await test('Explicit rejection allows retry and removes pending job',async()=>{const {ctx}=environment();ctx.fetch=async()=>json({error:'Another operation is in progress'},409);const b=button(),s={};await ctx.startUpdate('app',b,s,0);assert.equal(b.textContent,'Retry');assert.equal(b.disabled,false);assert.equal(ctx.pendingUpdates().app,undefined);assert(s.textContent.includes('Another operation'))});
 await test('Unknown job offers Check, never blind Retry',async()=>{const {ctx}=environment();ctx.fetch=async()=>json({status:'unknown',error:'Verify container state'});const b=button(),s={};await ctx.waitForUpdate('app',{id:'job12345',started:Date.now()},b,s,0);assert.equal(b.dataset.action,'check');assert.equal(b.textContent,'Check')});
 await test('Completed job clears persistent tracking',async()=>{const {ctx}=environment();ctx.rememberUpdate('app',{id:'job12345'});ctx.fetch=async()=>json({status:'completed'});const b=button(),s={};await ctx.waitForUpdate('app',{id:'job12345',started:Date.now()},b,s,0);assert(b.removed);assert.equal(s.textContent,'Updated');assert.equal(ctx.pendingUpdates().app,undefined)});
 await test('Concrete backend errors reach interface',async()=>{const {ctx}=environment();ctx.fetch=async()=>json({status:'error',error:'Original Compose file unavailable'});const b=button(),s={};await ctx.waitForUpdate('app',{id:'job12345',started:Date.now()},b,s,0);assert.equal(s.textContent,'Original Compose file unavailable')});
 await test('Detached row does not continue polling',async()=>{const {ctx}=environment();let calls=0;ctx.fetch=async()=>{calls++;return json({status:'running'})};const b=button();b.isConnected=false;await ctx.waitForUpdate('app',{id:'job12345',started:Date.now()},b,{},0);assert.equal(calls,0)});
 await test('Externally sourced text escaped',async()=>{const {ctx}=environment();assert.equal(ctx.escapeHTML('<img "x">'), '&lt;img &quot;x&quot;&gt;')});
 const app=fs.readFileSync('web/static/js/app.js','utf8');assert(!app.includes('async function loadUpdates'));assert(!app.includes('DOMContentLoaded, loadUpdates'));console.log('PASS one maintenance scan implementation');
 console.log(`${tests.length+1} frontend tests passed`);
})().catch(e=>{console.error(e);process.exitCode=1});
