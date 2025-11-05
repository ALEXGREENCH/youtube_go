FFmpeg build with 3GP support:

./configure
--prefix=/usr/local
--enable-gpl
--enable-nonfree
--enable-version3
--enable-libopencore-amrnb
--enable-libopencore-amrwb
--enable-decoder=h264
--enable-parser=h264
--enable-decoder=h263
--enable-encoder=h263
--enable-decoder=amrnb
--enable-decoder=amrwb
--enable-encoder=libopencore_amrnb
--enable-muxer=3gp
--enable-demuxer=mov
--enable-protocol=http
--enable-protocol=file
--enable-protocol=pipe
--enable-encoder=aac --enable-decoder=aac --enable-muxer=mp4
--enable-muxer=mov --enable-muxer=mp4 --enable-muxer=3gp
--enable-small

sudo make install
