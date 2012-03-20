<xsl:stylesheet xmlns:xsl = "http://www.w3.org/1999/XSL/Transform" version = "1.0" >
<!--<xsl:output omit-xml-declaration="no" method="xml" doctype-public="-//W3C//DTD XHTML 1.0 Strict//EN" doctype-system="http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd" indent="yes" encoding="UTF-8" />-->
<xsl:output omit-xml-declaration="no" method="xml" indent="yes" encoding="UTF-8" />
<xsl:template match = "/icestats" >
<mounts>
<xsl:for-each select="source">
<xsl:choose>
<xsl:when test="listeners">
<mount>
<point><xsl:value-of select="@mount" /></point>
<xsl:if test="server_name">
<name><xsl:value-of select="server_name" /></name>
</xsl:if>
<xsl:if test="server_description">
<description> <xsl:value-of select="server_description" /></description>
</xsl:if>
<xsl:if test="server_type">
<type><xsl:value-of select="server_type" /></type>
</xsl:if>
<xsl:if test="stream_start">
<start><xsl:value-of select="stream_start" /></start>
</xsl:if>
<xsl:if test="bitrate">
<bitrate><xsl:value-of select="bitrate" /></bitrate>
</xsl:if>
<xsl:if test="quality">
<quality><xsl:value-of select="quality" /></quality>
</xsl:if>
<xsl:if test="listeners">
<listeners><xsl:value-of select="listeners" /></listeners>
</xsl:if>
<xsl:if test="listener_peak">
<peak><xsl:value-of select="listener_peak" /></peak>
</xsl:if>
<xsl:if test="genre">
<genre><xsl:value-of select="genre" /></genre>
</xsl:if>
<xsl:if test="server_url">
<url><xsl:value-of select="server_url" /></url>
</xsl:if>
<xsl:if test="artist">
<artist><xsl:value-of select="artist" /></artist>
</xsl:if>
<title><xsl:value-of select="title" /></title>
</mount>
</xsl:when>
</xsl:choose>
</xsl:for-each>
</mounts>
</xsl:template>
</xsl:stylesheet>
