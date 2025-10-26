package ui

import "strings"

// AppendSuggestionScript injects the datalist script.
func AppendSuggestionScript(b *strings.Builder) {
	b.WriteString(`<datalist id="suggestions"></datalist>
<script>
(function(){
 var input = document.getElementById('search-input');
 var list = document.getElementById('suggestions');
 if(!input || !list){return;}
 var xhr = null;
 input.addEventListener('input', function(){
  var value = input.value;
  if(!value){ value = ''; }
  value = value.replace(/^\s+|\s+$/g,'');
  if(xhr){
   try{ xhr.onreadystatechange = null; xhr.abort(); }catch(ignore){}
  }
  if(value.length < 2){
   list.innerHTML = '';
   return;
  }
  xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function(){
   if(xhr.readyState !== 4){
    return;
   }
   if(xhr.status !== 200){
    return;
   }
   var payload;
   try{
    payload = JSON.parse(xhr.responseText);
   }catch(e){
    return;
   }
   var suggestions = payload && payload.suggestions;
   if(!suggestions || !suggestions.length){
    return;
   }
   var html = '';
   for(var i=0;i<suggestions.length;i++){
    var opt = suggestions[i];
    if(typeof opt !== 'string'){ opt = String(opt); }
    html += '<option value="' + opt.replace(/"/g,'&quot;') + '"></option>';
   }
   list.innerHTML = html;
  };
  xhr.open('GET','/suggest?q=' + encodeURIComponent(value), true);
  try{
   xhr.send(null);
  }catch(e){}
 });
})();
</script>
`)
}
