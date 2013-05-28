$("section").eq(0).after($("<div id='disqus_thread'><a href='#'>comments?</a></div>"))
$("#disqus_thread").click(function(e){
	e.preventDefault()
	var disqus = "<script>"
				+ "var disqus_shortname = 'zhanglin';"
				+ "(function() {"
				+ "var dsq = document.createElement('script'); dsq.type = 'text/javascript'; dsq.async = true;"
				+ "dsq.src = '//' + disqus_shortname + '.disqus.com/embed.js';"
				+ "(document.getElementsByTagName('head')[0] || document.getElementsByTagName('body')[0]).appendChild(dsq);"
				+ "})();"
				+ "</script>"
	$(this).append($(disqus))
})