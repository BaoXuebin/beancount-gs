(this["webpackJsonpbeancount-web"]=this["webpackJsonpbeancount-web"]||[]).push([[13],{461:function(t,e,n){"use strict";n.r(e);var i=n(11),a=n(12),r=n(14),o=n(13),c=n(466),s=n(455),u=n(203),h=n(451),l=n(350),d=(n(445),n(0)),f=n(63),b=n(15),p=n(81),j=n(1),g=function(t){Object(r.a)(n,t);var e=Object(o.a)(n);function n(){var t;Object(i.a)(this,n);for(var a=arguments.length,r=new Array(a),o=0;o<a;o++)r[o]=arguments[o];return(t=e.call.apply(e,[this].concat(r))).theme=t.context.theme,t.state={loading:!1,path:null,files:[],rawContent:"",content:"",editorState:l.EditorState.createEmpty()},t.fetchFileDir=function(){t.setState({loading:!0}),Object(f.b)("/api/auth/file/dir").then((function(e){t.setState({files:e})})).finally((function(){t.setState({loading:!1})}))},t.handldEditContent=function(e){t.setState({editorState:e,content:e.getCurrentContent().getPlainText()})},t.handleChangeFile=function(e){t.setState({path:e},(function(){t.fetchFileContent(e)}))},t.fetchFileContent=function(){t.setState({loading:!0}),Object(f.b)("/api/auth/file/content?path=".concat(t.state.path)).then((function(e){t.setState({rawContent:e,content:e,editorState:l.EditorState.createWithContent(l.ContentState.createFromText(e))})})).finally((function(){t.setState({loading:!1})}))},t.saveFileContent=function(){var e=t.state,n=e.path,i=e.content;t.setState({loading:!0}),Object(f.b)("/api/auth/file",{method:"POST",body:{path:n,content:i}}).then((function(){t.setState({rawContent:i}),s.b.success("\u4fdd\u5b58\u6210\u529f")})).finally((function(){t.setState({loading:!1})}))},t}return Object(a.a)(n,[{key:"componentDidMount",value:function(){this.fetchFileDir()}},{key:"render",value:function(){return this.context.theme!==this.theme&&(this.theme=this.context.theme),Object(j.jsxs)("div",{className:"edit-page",children:[Object(j.jsxs)("div",{children:[Object(j.jsx)(u.a,{showSearch:!0,placeholder:"\u8bf7\u9009\u62e9\u6e90\u6587\u4ef6",style:{width:"200px"},onChange:this.handleChangeFile,children:this.state.files.map((function(t){return Object(j.jsx)(u.a.Option,{value:t,children:t},t)}))}),"\xa0\xa0",Object(j.jsx)(h.a,{type:"primary",icon:Object(j.jsx)(c.a,{}),disabled:this.state.rawContent===this.state.content,loading:this.state.loading,onClick:this.saveFileContent,children:"\u4fdd\u5b58"})]}),Object(j.jsx)("div",{style:{marginTop:"1rem"},children:Object(j.jsx)(l.Editor,{placeholder:"\u672a\u9009\u62e9\u6587\u4ef6",editorState:this.state.editorState,onChange:this.handldEditContent})})]})}}]),n}(d.Component);g.contextType=b.a,e.default=Object(p.a)(g)},63:function(t,e,n){"use strict";n.d(e,"c",(function(){return s})),n.d(e,"d",(function(){return u})),n.d(e,"e",(function(){return h})),n.d(e,"a",(function(){return l})),n.d(e,"b",(function(){return f})),n.d(e,"f",(function(){return b}));var i=n(455),a=n(89),r=n.n(a),o=n(212),c=n.n(o),s=function(t){var e=t.split(":");return e&&e.length>=1?t.split(":")[0]:""},u=function(t){for(var e=t.split(":").reverse(),n=0;n<e.length;n++){var i=e[n];if(new RegExp("^[a-zA-Z]").test(i))return i}return""},h=function(t){var e=t.split(":");return e&&e.length>=2?t.split(":")[e.length-1]:""},l={Income:"\u6536\u5165",Expenses:"\u652f\u51fa",Liabilities:"\u8d1f\u503a",Assets:"\u8d44\u4ea7"},d=function(t){return t},f=function(t){var e=arguments.length>1&&void 0!==arguments[1]?arguments[1]:{},n=e.method,a=e.headers,r=e.body,o={"Content-Type":"application/json",ledgerId:window.localStorage.getItem("ledgerId")};return new Promise((function(e,s){c()(t,{method:n,headers:Object.assign({},o,a),body:JSON.stringify(r)}).then(d).then((function(t){return t.json()})).then((function(t){var n=t.code;200===n?e(t.data):200!==n&&(400===n?i.b.error("\u8bf7\u6c42\u53c2\u6570\u9519\u8bef"):1001===n?i.b.error("\u8d26\u76ee\u4e0d\u5e73\u8861"):1003===n?i.b.error("\u65e0\u6548\u8d26\u6237"):1005===n?i.b.error("\u65e0\u6548\u547d\u4ee4"):1006===n?i.b.error("\u5bc6\u7801\u9519\u8bef"):1010===n?window.location.href="/#/ledger":i.b.error("\u8bf7\u6c42\u5931\u8d25\uff0c\u8bf7\u5237\u65b0\u9875\u9762\u91cd\u8bd5"),s(t))})).catch((function(t){i.b.error("\u8bf7\u6c42\u5931\u8d25\uff0c\u8bf7\u5237\u65b0\u9875\u9762\u91cd\u8bd5"),s(t)}))}))},b=function(){return r()().format("YYYY-M")}},81:function(t,e,n){"use strict";var i=n(88),a=n(11),r=n(12),o=n(14),c=n(13),s=n(0),u=n(1);e.a=function(t){return function(e){Object(o.a)(s,e);var n=Object(c.a)(s);function s(){return Object(a.a)(this,s),n.apply(this,arguments)}return Object(r.a)(s,[{key:"render",value:function(){return Object(u.jsx)(t,Object(i.a)({},this.props))}}]),s}(s.Component)}}}]);