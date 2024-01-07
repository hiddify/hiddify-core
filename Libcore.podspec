Pod::Spec.new do |s|    
    s.name             = 'Libcore'
    s.version          = '0.10.0'
    s.summary          = 'hiddify mobile SDK for iOS'  
    s.homepage         = 'https://hiddify.com/'
    s.license          = { :type => 'Copyright', :text => 'Hiddify Open Software' }
    s.author           = { 'Hiddify' => 'ios@hiddifyk.com' }
    
    s.ios.deployment_target = '9.0'
    s.vendored_frameworks = 'Libcore.framework'
    s.source           = { :path => '../libcore/' }


  end
  
